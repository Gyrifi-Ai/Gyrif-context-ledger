package tests

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type memoryTarget struct {
	mu        sync.Mutex
	values    map[string]json.RawMessage
	failApply bool
}

func (target *memoryTarget) Read(_ context.Context, unit string) (targets.Value, error) {
	target.mu.Lock()
	defer target.mu.Unlock()
	value, exists := target.values[unit]
	copyValue := append(json.RawMessage(nil), value...)
	return targets.Value{Unit: unit, Value: copyValue, Fingerprint: func() string {
		if !exists {
			return ""
		}
		return ledger.Fingerprint(copyValue)
	}(), Exists: exists}, nil
}
func (target *memoryTarget) Fingerprint(ctx context.Context, unit string) (string, error) {
	value, err := target.Read(ctx, unit)
	return value.Fingerprint, err
}
func (target *memoryTarget) Preview(_ context.Context, changes []ledger.Change) (targets.Preview, error) {
	return targets.Preview{Fidelity: "REFERENCE", Summary: "in-memory preview"}, nil
}
func (target *memoryTarget) Compile(_ context.Context, changes []ledger.Change) (targets.Plan, error) {
	plan := targets.Plan{}
	for _, change := range changes {
		plan.Operations = append(plan.Operations, targets.Operation{Unit: change.Unit, Action: change.Action, Desired: change.Desired, DesiredFingerprint: change.DesiredFingerprint})
	}
	return plan, nil
}
func (target *memoryTarget) Apply(_ context.Context, plan targets.Plan) error {
	target.mu.Lock()
	defer target.mu.Unlock()
	if target.failApply {
		return errors.New("injected apply failure")
	}
	for _, operation := range plan.Operations {
		if operation.Action == ledger.ChangeDelete {
			delete(target.values, operation.Unit)
		} else {
			target.values[operation.Unit] = append(json.RawMessage(nil), operation.Desired...)
		}
	}
	return nil
}
func (target *memoryTarget) Verify(ctx context.Context, plan targets.Plan) error {
	for _, operation := range plan.Operations {
		value, _ := target.Read(ctx, operation.Unit)
		if operation.Action == ledger.ChangeDelete && value.Exists {
			return errors.New("delete failed")
		}
		if operation.Action == ledger.ChangePut && value.Fingerprint != operation.DesiredFingerprint {
			return errors.New("put failed")
		}
	}
	return nil
}
func (target *memoryTarget) Restore(ctx context.Context, plan targets.Plan) error {
	return target.Apply(ctx, plan)
}
func (target *memoryTarget) Capabilities() targets.Capabilities {
	return targets.Capabilities{AtomicApply: true, ExactPreview: true, ConditionalWrite: true, Batch: true, Restore: true}
}

func newEngine(t *testing.T, target *memoryTarget) (*engine.Engine, string) {
	t.Helper()
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(context.Background(), filepath.Join(directory, "state.db"), filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	application := engine.New(repo, target, nil)
	ledgerValue, err := application.CreateLedger(context.Background(), "Test ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	return application, ledgerValue.ID
}

func TestChangeProposalEvaluationApprovalReleaseFlow(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}}
	application, ledgerID := newEngine(t, target)
	request := engine.CreateChangeRequest{Unit: "42", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":42,"payload":{"answer":"yes"}}`), IdempotencyKey: "request-1"}
	change, err := application.CreateChange(ctx, ledgerID, request)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := application.CreateChange(ctx, ledgerID, request)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != change.ID {
		t.Fatal("idempotent request created another Change")
	}
	request.Desired = json.RawMessage(`{"id":42,"payload":{"answer":"no"}}`)
	if _, err := application.CreateChange(ctx, ledgerID, request); err == nil {
		t.Fatal("idempotency key reuse should conflict")
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Answer update", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "must be safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	release, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if release.ParentID != "" {
		t.Fatal("first release must have empty parent")
	}
	second, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "42", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":42,"payload":{"answer":"no"}}`), IdempotencyKey: "request-2"})
	if err != nil {
		t.Fatal(err)
	}
	secondProposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Second update", ChangeIDs: []string{second.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = application.EvaluateProposal(ctx, ledgerID, secondProposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err = application.ApproveProposal(ctx, ledgerID, secondProposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err = application.ReleaseProposal(ctx, ledgerID, secondProposal.ID); err != nil {
		t.Fatal(err)
	}
	rollbackProposal, err := application.CreateRollbackProposal(ctx, ledgerID, release.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = application.EvaluateProposal(ctx, ledgerID, rollbackProposal.ID, "restore"); err != nil {
		t.Fatal(err)
	}
	if err = application.ApproveProposal(ctx, ledgerID, rollbackProposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	rollbackRelease, err := application.ReleaseProposal(ctx, ledgerID, rollbackProposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rollbackRelease.ParentID == release.ID {
		t.Fatal("rollback rewound HEAD instead of creating forward history")
	}
	value, _ := target.Read(ctx, "42")
	if string(value.Value) != `{"id":42,"payload":{"answer":"yes"}}` {
		t.Fatalf("rollback state mismatch: %s", value.Value)
	}
	history, err := application.ListReleases(ctx, ledgerID)
	if err != nil || len(history) != 3 {
		t.Fatalf("release history mismatch: %v %v", history, err)
	}
}

func TestTargetApplyFailureLeavesUnfinishedIntent(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}, failApply: true}
	application, ledgerID := newEngine(t, target)
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "7", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":7}`), IdempotencyKey: "fail"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Failure", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err == nil {
		t.Fatal("release should fail")
	}
	releases, err := application.ListReleases(ctx, ledgerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 0 {
		t.Fatal("HEAD history advanced after apply failure")
	}
}
