package tests

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type memoryTarget struct {
	mu         sync.Mutex
	values     map[string]json.RawMessage
	failApply  bool
	failVerify bool
	failRead   bool
	applyCalls int
	readCalls  map[string]int
}

func (target *memoryTarget) Read(_ context.Context, unit string) (targets.Value, error) {
	target.mu.Lock()
	defer target.mu.Unlock()
	if target.readCalls != nil {
		target.readCalls[unit]++
	}
	if target.failRead {
		return targets.Value{}, errors.New("injected read failure")
	}
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
	target.applyCalls++
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
	if target.failVerify {
		return errors.New("injected verify failure")
	}
	mismatches := make([]targets.VerificationMismatch, 0)
	for _, operation := range plan.Operations {
		value, err := target.Read(ctx, operation.Unit)
		if err != nil {
			return err
		}
		if operation.Action == ledger.ChangeDelete && value.Exists {
			mismatches = append(mismatches, targets.VerificationMismatch{Unit: operation.Unit, Expected: operation.DesiredFingerprint, Observed: value.Fingerprint})
		}
		if operation.Action == ledger.ChangePut && value.Fingerprint != operation.DesiredFingerprint {
			mismatches = append(mismatches, targets.VerificationMismatch{Unit: operation.Unit, Expected: operation.DesiredFingerprint, Observed: value.Fingerprint})
		}
	}
	if len(mismatches) != 0 {
		return &targets.VerificationError{Mismatches: mismatches}
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
	if err := application.StartPreparation(context.Background(), engine.PreparationOptions{BatchSize: 25, Lease: 50 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	ledgerValue, err := application.CreateLedger(context.Background(), "Test ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	return application, ledgerValue.ID
}

func waitForChangeStatus(t *testing.T, application *engine.Engine, ledgerID, changeID string, status ledger.ChangeStatus) ledger.Change {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		page, err := application.ListChanges(context.Background(), ledgerID, engine.ListRequest{Limit: 200, Status: string(status)})
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range page.Items {
			if change.ID == changeID {
				return change
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("Change %s did not reach %s", changeID, status)
	return ledger.Change{}
}

func nextEvent(t *testing.T, events <-chan engine.Event) engine.Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("expected domain event")
		return engine.Event{}
	}
}

func TestChangeProposalEvaluationApprovalReleaseFlow(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}}
	application, ledgerID := newEngine(t, target)
	events, unsubscribe := application.Events().Subscribe(16)
	defer unsubscribe()
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
	waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
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
	wantKinds := []engine.EventKind{
		engine.EventChangeAccepted,
		engine.EventChangeReady,
		engine.EventProposalCreated,
		engine.EventProposalEvaluated,
		engine.EventProposalApproved,
		engine.EventReleaseStarted,
		engine.EventReleaseCompleted,
	}
	wantSubjects := []string{change.ID, change.ID, proposal.ID, proposal.ID, proposal.ID, "intent_", release.ID}
	for index, wantKind := range wantKinds {
		event := nextEvent(t, events)
		if event.Kind != wantKind || event.LedgerID != ledgerID {
			t.Fatalf("event %d = %#v, want kind %s for ledger %s", index, event, wantKind, ledgerID)
		}
		if index == 5 {
			if !strings.HasPrefix(event.SubjectID, wantSubjects[index]) {
				t.Fatalf("release.started subject = %q", event.SubjectID)
			}
		} else if event.SubjectID != wantSubjects[index] {
			t.Fatalf("event %d subject = %q, want %q", index, event.SubjectID, wantSubjects[index])
		}
		if event.At.IsZero() {
			t.Fatalf("event %d has zero timestamp", index)
		}
	}
	select {
	case event := <-events:
		t.Fatalf("unexpected event after idempotent replay: %#v", event)
	default:
	}
	second, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "42", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":42,"payload":{"answer":"no"}}`), IdempotencyKey: "request-2"})
	if err != nil {
		t.Fatal(err)
	}
	waitForChangeStatus(t, application, ledgerID, second.ID, ledger.ChangeReady)
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
	history, err := application.ListReleases(ctx, ledgerID, engine.ListRequest{})
	if err != nil || len(history.Items) != 3 {
		t.Fatalf("release history mismatch: %v %v", history.Items, err)
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
	waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
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
	events, unsubscribe := application.Events().Subscribe(4)
	defer unsubscribe()
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err == nil {
		t.Fatal("release should fail")
	}
	if event := nextEvent(t, events); event.Kind != engine.EventReleaseStarted {
		t.Fatalf("first failure event = %s", event.Kind)
	}
	if event := nextEvent(t, events); event.Kind != engine.EventReleaseFailed {
		t.Fatalf("second failure event = %s", event.Kind)
	}
	releases, err := application.ListReleases(ctx, ledgerID, engine.ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(releases.Items) != 0 {
		t.Fatal("HEAD history advanced after apply failure")
	}
	recoveryEvents, unsubscribeRecovery := application.Events().Subscribe(1)
	defer unsubscribeRecovery()
	if err := application.RecoverReleases(ctx); err != nil {
		t.Fatal(err)
	}
	if event := nextEvent(t, recoveryEvents); event.Kind != engine.EventIntentRecoveryRequired {
		t.Fatalf("recovery event = %s", event.Kind)
	}
}

func TestTargetVerifyFailurePublishesReleaseFailed(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}, failVerify: true}
	application, ledgerID := newEngine(t, target)
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "8", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":8}`), IdempotencyKey: "verify-fail"})
	if err != nil {
		t.Fatal(err)
	}
	waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Verification failure", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := application.Events().Subscribe(2)
	defer unsubscribe()
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err == nil {
		t.Fatal("release should fail verification")
	}
	if event := nextEvent(t, events); event.Kind != engine.EventReleaseStarted {
		t.Fatalf("first failure event = %s", event.Kind)
	}
	if event := nextEvent(t, events); event.Kind != engine.EventReleaseFailed {
		t.Fatalf("second failure event = %s", event.Kind)
	}
}
