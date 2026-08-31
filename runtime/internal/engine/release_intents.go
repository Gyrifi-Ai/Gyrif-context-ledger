package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type ReleaseIntentOperation struct {
	targets.Operation
	HasBeforeImage bool `json:"hasBeforeImage"`
}

type ReleaseIntentPlan struct {
	Operations []ReleaseIntentOperation `json:"operations"`
}

type ReleaseIntent struct {
	ID             string                     `json:"id"`
	LedgerID       string                     `json:"ledgerId"`
	ProposalID     string                     `json:"proposalId"`
	ProposalHash   string                     `json:"proposalHash"`
	ParentID       string                     `json:"parentId,omitempty"`
	Status         ledger.ReleaseIntentStatus `json:"status"`
	Plan           ReleaseIntentPlan          `json:"plan"`
	CreatedAt      time.Time                  `json:"createdAt"`
	Resolution     string                     `json:"resolution,omitempty"`
	ResolutionNote string                     `json:"resolutionNote,omitempty"`
	ResolvedAt     *time.Time                 `json:"resolvedAt,omitempty"`
}

type RetryReleaseIntentResult struct {
	Resolved   bool                           `json:"resolved"`
	Mismatches []targets.VerificationMismatch `json:"mismatches"`
}

func ValidReleaseIntentStatus(status ledger.ReleaseIntentStatus) bool {
	switch status {
	case ledger.IntentReady, ledger.IntentApplying, ledger.IntentVerifying, ledger.IntentFinalized, ledger.IntentRecoveryRequired, ledger.IntentAbandoned:
		return true
	default:
		return false
	}
}

func (engine *Engine) releaseIntentView(ctx context.Context, intent ledger.ReleaseIntent) (ReleaseIntent, error) {
	var plan targets.Plan
	if err := json.Unmarshal(intent.Plan, &plan); err != nil {
		return ReleaseIntent{}, wrap(CodeInternal, "Release Intent plan is unreadable.", err)
	}
	operations := make([]ReleaseIntentOperation, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		hasBeforeImage := false
		if operation.BeforeObjectHash != "" {
			_, err := engine.repository.ReadObject(ctx, operation.BeforeObjectHash)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return ReleaseIntent{}, wrap(CodeInternal, "Could not inspect retained before-image.", err)
			}
			hasBeforeImage = err == nil
		}
		operations = append(operations, ReleaseIntentOperation{Operation: operation, HasBeforeImage: hasBeforeImage})
	}
	return ReleaseIntent{
		ID:             intent.ID,
		LedgerID:       intent.LedgerID,
		ProposalID:     intent.ProposalID,
		ProposalHash:   intent.ProposalHash,
		ParentID:       intent.ParentID,
		Status:         intent.Status,
		Plan:           ReleaseIntentPlan{Operations: operations},
		CreatedAt:      intent.CreatedAt,
		Resolution:     intent.Resolution,
		ResolutionNote: intent.ResolutionNote,
		ResolvedAt:     intent.ResolvedAt,
	}, nil
}

func (engine *Engine) loadScopedReleaseIntent(ctx context.Context, ledgerID, intentID string) (ledger.ReleaseIntent, error) {
	intent, err := engine.repository.LoadReleaseIntent(ctx, intentID)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && intent.LedgerID != ledgerID) {
		return ledger.ReleaseIntent{}, wrap(CodeNotFound, "Release Intent was not found.", repository.ErrNotFound)
	}
	if err != nil {
		return ledger.ReleaseIntent{}, wrap(CodeInternal, "Could not load Release Intent.", err)
	}
	return intent, nil
}

func (engine *Engine) LoadReleaseIntent(ctx context.Context, ledgerID, intentID string) (ReleaseIntent, error) {
	intent, err := engine.loadScopedReleaseIntent(ctx, ledgerID, intentID)
	if err != nil {
		return ReleaseIntent{}, err
	}
	return engine.releaseIntentView(ctx, intent)
}

func (engine *Engine) ListReleaseIntents(ctx context.Context, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ReleaseIntent, error) {
	if status != nil && !ValidReleaseIntentStatus(*status) {
		return nil, wrap(CodeInvalid, "Release Intent status is invalid.", ledger.ErrInvalid)
	}
	items, err := engine.repository.ListReleaseIntentsForLedger(ctx, ledgerID, status)
	if err != nil {
		return nil, wrap(CodeInternal, "Could not load Release Intents.", err)
	}
	views := make([]ReleaseIntent, 0, len(items))
	for _, item := range items {
		view, err := engine.releaseIntentView(ctx, item)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (engine *Engine) RetryReleaseIntent(ctx context.Context, ledgerID, intentID string) (RetryReleaseIntentResult, error) {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	intent, err := engine.loadScopedReleaseIntent(ctx, ledgerID, intentID)
	if err != nil {
		return RetryReleaseIntentResult{}, err
	}
	if intent.Status != ledger.IntentRecoveryRequired && intent.Status != ledger.IntentVerifying {
		return RetryReleaseIntentResult{}, wrap(CodeConflict, fmt.Sprintf("Release Intent is %s and cannot be retried.", intent.Status), ledger.ErrConflict)
	}
	var plan targets.Plan
	if err := json.Unmarshal(intent.Plan, &plan); err != nil {
		return RetryReleaseIntentResult{}, wrap(CodeInternal, "Release Intent plan is unreadable.", err)
	}
	if err := engine.target.Verify(ctx, plan); err != nil {
		var verificationError *targets.VerificationError
		if errors.As(err, &verificationError) {
			if intent.Status != ledger.IntentRecoveryRequired {
				if updateErr := engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentRecoveryRequired); updateErr != nil {
					return RetryReleaseIntentResult{}, wrap(CodeInternal, "Could not update Release Intent recovery state.", updateErr)
				}
				engine.publish(EventIntentRecoveryRequired, ledgerID, intent.ID)
			}
			mismatches := append([]targets.VerificationMismatch(nil), verificationError.Mismatches...)
			return RetryReleaseIntentResult{Mismatches: mismatches}, nil
		}
		return RetryReleaseIntentResult{}, wrap(CodeUnavailable, "Target is unavailable during Release recovery.", err)
	}
	if _, err := engine.finalizeIntent(ctx, intent); err != nil {
		return RetryReleaseIntentResult{}, err
	}
	engine.publish(EventIntentResolved, ledgerID, intent.ID)
	return RetryReleaseIntentResult{Resolved: true, Mismatches: []targets.VerificationMismatch{}}, nil
}

func (engine *Engine) ResolveReleaseIntent(ctx context.Context, ledgerID, intentID, resolution, note string) error {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	intent, err := engine.loadScopedReleaseIntent(ctx, ledgerID, intentID)
	if err != nil {
		return err
	}
	if resolution != string(ledger.IntentAbandoned) {
		return wrap(CodeInvalid, "Resolution must be ABANDONED.", ledger.ErrInvalid)
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return wrap(CodeInvalid, "Resolution note is required.", ledger.ErrInvalid)
	}
	if intent.Status != ledger.IntentRecoveryRequired {
		return wrap(CodeConflict, fmt.Sprintf("Release Intent is %s and cannot be resolved.", intent.Status), ledger.ErrConflict)
	}
	if err := engine.repository.ResolveReleaseIntent(ctx, intent.ID, note, time.Now().UTC()); err != nil {
		if errors.Is(err, ledger.ErrConflict) {
			return wrap(CodeConflict, "Release Intent is no longer awaiting recovery.", err)
		}
		return wrap(CodeInternal, "Could not resolve Release Intent.", err)
	}
	engine.publish(EventIntentResolved, ledgerID, intent.ID)
	return nil
}
