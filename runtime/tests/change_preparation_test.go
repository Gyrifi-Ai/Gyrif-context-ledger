package tests

import (
	"context"
	"encoding/json"
	"fmt"
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

type semanticTarget struct{ *memoryTarget }

type blockingTarget struct {
	*memoryTarget
	once    sync.Once
	entered chan struct{}
}

func (target *blockingTarget) Read(ctx context.Context, _ string) (targets.Value, error) {
	target.once.Do(func() { close(target.entered) })
	<-ctx.Done()
	return targets.Value{}, ctx.Err()
}

func (target *semanticTarget) Prepare(context.Context, ledger.Change) (targets.Value, error) {
	return targets.Value{}, fmt.Errorf("%w: unsupported desired value", targets.ErrSemantic)
}

func newPreparationEngine(t *testing.T, target targets.TargetAdapter) (*engine.Engine, string) {
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
	value, err := application.CreateLedger(context.Background(), "Preparation", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	return application, value.ID
}

func TestChangePreparationOutcomes(t *testing.T) {
	t.Run("observed fingerprints and no-op", func(t *testing.T) {
		same := json.RawMessage(`{"id":"same"}`)
		different := json.RawMessage(`{"id":"different","old":true}`)
		target := &memoryTarget{values: map[string]json.RawMessage{"same": same, "different": different}}
		application, ledgerID := newPreparationEngine(t, target)
		tests := []struct {
			unit    string
			action  ledger.ChangeAction
			desired json.RawMessage
			base    string
			noop    bool
		}{
			{unit: "absent", action: ledger.ChangePut, desired: json.RawMessage(`{"id":"absent"}`)},
			{unit: "different", action: ledger.ChangePut, desired: json.RawMessage(`{"id":"different","new":true}`), base: ledger.Fingerprint(different)},
			{unit: "same", action: ledger.ChangePut, desired: same, base: ledger.Fingerprint(same), noop: true},
			{unit: "missing-delete", action: ledger.ChangeDelete, noop: true},
		}
		for index, test := range tests {
			change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: test.unit, Action: test.action, Desired: test.desired, IdempotencyKey: fmt.Sprintf("prepare-%d", index)})
			if err != nil {
				t.Fatal(err)
			}
			prepared := waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
			if prepared.BaseFingerprint != test.base || prepared.Noop != test.noop {
				t.Fatalf("prepared Change = %#v", prepared)
			}
		}
	})

	t.Run("semantic rejection is invalid", func(t *testing.T) {
		application, ledgerID := newPreparationEngine(t, &semanticTarget{&memoryTarget{values: map[string]json.RawMessage{}}})
		change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: "bad", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":"bad"}`), IdempotencyKey: "semantic"})
		if err != nil {
			t.Fatal(err)
		}
		invalid := waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeInvalid)
		if invalid.InvalidReason == "" {
			t.Fatal("semantic rejection did not persist a reason")
		}
		if _, err := application.CreateProposal(context.Background(), ledgerID, engine.CreateProposalRequest{Title: "invalid", ChangeIDs: []string{change.ID}}); err == nil || !strings.Contains(err.Error(), "INVALID") {
			t.Fatalf("invalid Proposal guard = %v", err)
		}
	})

	t.Run("no target remains usable", func(t *testing.T) {
		application, ledgerID := newPreparationEngine(t, nil)
		change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: "offline", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":"offline"}`), IdempotencyKey: "no-target"})
		if err != nil {
			t.Fatal(err)
		}
		prepared := waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
		if prepared.BaseFingerprint != "" {
			t.Fatalf("base fingerprint = %q", prepared.BaseFingerprint)
		}
	})
}

func TestPreparationRetriesWithoutInvalidating(t *testing.T) {
	target := &memoryTarget{values: map[string]json.RawMessage{}, failRead: true}
	application, ledgerID := newPreparationEngine(t, target)
	change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: "retry", Action: ledger.ChangeDelete, IdempotencyKey: "retry"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	accepted := waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeAccepted)
	if accepted.Status != ledger.ChangeAccepted {
		t.Fatalf("status = %s", accepted.Status)
	}
	target.mu.Lock()
	target.failRead = false
	target.mu.Unlock()
	waitForChangeStatus(t, application, ledgerID, change.ID, ledger.ChangeReady)
}

func TestPreparationShutdownCancelsTargetRead(t *testing.T) {
	target := &blockingTarget{memoryTarget: &memoryTarget{values: map[string]json.RawMessage{}}, entered: make(chan struct{})}
	application, ledgerID := newPreparationEngine(t, target)
	change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: "blocked", Action: ledger.ChangeDelete, IdempotencyKey: "blocked"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-target.entered:
	case <-time.After(time.Second):
		t.Fatal("preparation did not enter target read")
	}
	if _, err := application.CreateProposal(context.Background(), ledgerID, engine.CreateProposalRequest{Title: "accepted", ChangeIDs: []string{change.ID}}); err == nil || !strings.Contains(err.Error(), "ACCEPTED") {
		t.Fatalf("accepted Proposal guard = %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- application.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not join preparation worker")
	}
}

func TestPreparationProcessesConcurrentBatchExactlyOnce(t *testing.T) {
	target := &memoryTarget{values: map[string]json.RawMessage{}, readCalls: map[string]int{}}
	application, ledgerID := newPreparationEngine(t, target)
	var group sync.WaitGroup
	ids := make(chan string, 200)
	for index := range 200 {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			unit := fmt.Sprintf("unit-%03d", index)
			change, err := application.CreateChange(context.Background(), ledgerID, engine.CreateChangeRequest{Unit: unit, Action: ledger.ChangeDelete, IdempotencyKey: unit})
			if err != nil {
				t.Errorf("CreateChange: %v", err)
				return
			}
			ids <- change.ID
		}(index)
	}
	group.Wait()
	close(ids)
	for id := range ids {
		waitForChangeStatus(t, application, ledgerID, id, ledger.ChangeReady)
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	if len(target.readCalls) != 200 {
		t.Fatalf("prepared units = %d", len(target.readCalls))
	}
	for unit, calls := range target.readCalls {
		if calls != 1 {
			t.Fatalf("%s read %d times", unit, calls)
		}
	}
}
