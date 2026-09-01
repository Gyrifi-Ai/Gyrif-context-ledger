package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type PreparationOptions struct {
	BatchSize int
	Lease     time.Duration
}

const preparationMaxAttempts = 10

type preparationWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (engine *Engine) StartPreparation(parent context.Context, options PreparationOptions) error {
	if options.BatchSize < 1 || options.Lease <= 0 {
		return fmt.Errorf("invalid preparation worker options")
	}
	engine.preparationMu.Lock()
	defer engine.preparationMu.Unlock()
	if engine.preparation != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	worker := &preparationWorker{cancel: cancel, done: make(chan struct{})}
	engine.preparation = worker
	go func() {
		defer close(worker.done)
		engine.runPreparation(ctx, options)
	}()
	engine.wakePreparation()
	return nil
}

func (engine *Engine) StopPreparation() {
	engine.preparationMu.Lock()
	worker := engine.preparation
	engine.preparation = nil
	engine.preparationMu.Unlock()
	if worker != nil {
		worker.cancel()
		<-worker.done
	}
}

func (engine *Engine) wakePreparation() {
	select {
	case engine.preparationWake <- struct{}{}:
	default:
	}
}

func (engine *Engine) runPreparation(ctx context.Context, options PreparationOptions) {
	ticker := time.NewTicker(min(options.Lease/2, 30*time.Second))
	defer ticker.Stop()
	owner, err := ledger.NewID("prep")
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-engine.preparationWake:
		case <-ticker.C:
		}
		for engine.prepareBatch(ctx, owner, options) == options.BatchSize {
		}
	}
}

func (engine *Engine) prepareBatch(ctx context.Context, owner string, options PreparationOptions) int {
	now := time.Now().UTC()
	changes, err := engine.repository.ClaimChangesForPreparation(ctx, owner, options.BatchSize, preparationMaxAttempts, now, now.Add(-options.Lease))
	if err != nil {
		return 0
	}
	for _, change := range changes {
		engine.prepareChange(ctx, owner, change)
	}
	return len(changes)
}

func (engine *Engine) prepareChange(ctx context.Context, owner string, change ledger.Change) {
	if engine.target == nil {
		engine.noTargetWarning.Do(func() { slog.Warn("Change preparation has no target; base fingerprints will be empty") })
		outcome := ledger.PrepareChangeOutcome(change, "", false)
		committed, _ := engine.repository.CompleteChangePreparation(ctx, repository.PreparationUpdate{LedgerID: change.LedgerID, ChangeID: change.ID, Owner: owner, Status: outcome.Status, Noop: outcome.Noop})
		if committed {
			engine.publish(EventChangeReady, change.LedgerID, change.ID)
		}
		return
	}
	var observed targets.Value
	var err error
	if preparer, ok := engine.targetHealth.(targets.ChangePreparer); ok {
		observed, err = preparer.Prepare(ctx, change)
	} else {
		observed, err = engine.target.Read(ctx, change.Unit)
	}
	if err != nil {
		if errors.Is(err, targets.ErrSemantic) {
			committed, _ := engine.repository.CompleteChangePreparation(ctx, repository.PreparationUpdate{LedgerID: change.LedgerID, ChangeID: change.ID, Owner: owner, Status: ledger.ChangeInvalid, InvalidReason: err.Error()})
			if committed {
				engine.publish(EventChangeInvalid, change.LedgerID, change.ID)
			}
			return
		}
		engine.retryPreparation(ctx, owner, change)
		return
	}
	outcome := ledger.PrepareChangeOutcome(change, observed.Fingerprint, observed.Exists)
	committed, _ := engine.repository.CompleteChangePreparation(ctx, repository.PreparationUpdate{LedgerID: change.LedgerID, ChangeID: change.ID, Owner: owner, Status: outcome.Status, BaseFingerprint: outcome.BaseFingerprint, Noop: outcome.Noop})
	if committed {
		engine.publish(EventChangeReady, change.LedgerID, change.ID)
	}
}

func (engine *Engine) retryPreparation(ctx context.Context, owner string, change ledger.Change) {
	delay := time.Second << min(change.PreparationAttempts, 8)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	retryAfter := time.Now().UTC().Add(delay)
	_, _ = engine.repository.CompleteChangePreparation(ctx, repository.PreparationUpdate{LedgerID: change.LedgerID, ChangeID: change.ID, Owner: owner, RetryAfter: &retryAfter})
}
