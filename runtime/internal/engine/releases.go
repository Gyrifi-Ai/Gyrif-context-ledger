package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

func (engine *Engine) ReleaseProposal(ctx context.Context, ledgerID, proposalID string) (ledger.Release, error) {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return ledger.Release{}, wrap(CodeNotFound, "Proposal was not found.", err)
	}
	passing, err := engine.repository.HasPassingCheck(ctx, proposal.ID, proposal.Hash)
	if err != nil {
		return ledger.Release{}, wrap(CodeInternal, "Could not verify checks.", err)
	}
	approved, err := engine.repository.HasApproval(ctx, proposal.ID, proposal.Hash)
	if err != nil {
		return ledger.Release{}, wrap(CodeInternal, "Could not verify approvals.", err)
	}
	if !passing || !approved {
		return ledger.Release{}, wrap(CodeConflict, "Current passing evidence and approval are required.", ledger.ErrReleaseNotReady)
	}
	head, err := engine.repository.CurrentHead(ctx, ledgerID)
	if err != nil {
		return ledger.Release{}, wrap(CodeInternal, "Could not load HEAD.", err)
	}
	if head.ReleaseID != proposal.BaseReleaseID {
		return ledger.Release{}, wrap(CodeConflict, "Ledger HEAD moved after this Proposal was created.", ledger.ErrConflict)
	}
	changes, err := engine.repository.LoadChanges(ctx, ledgerID, proposal.ChangeIDs)
	if err != nil {
		return ledger.Release{}, wrap(CodeInternal, "Could not load release Changes.", err)
	}
	plan, err := engine.target.Compile(ctx, changes)
	if err != nil {
		return ledger.Release{}, wrap(CodeUnavailable, "Target could not compile the release.", err)
	}
	for index := range plan.Operations {
		current, err := engine.target.Read(ctx, plan.Operations[index].Unit)
		if err != nil {
			return ledger.Release{}, wrap(CodeUnavailable, "Could not capture target before-image.", err)
		}
		plan.Operations[index].ExpectedFingerprint = current.Fingerprint
		plan.Operations[index].ExpectedExists = current.Exists
		plan.Operations[index].BeforeExists = current.Exists
		if current.Exists {
			objectHash, err := engine.repository.WriteObject(ctx, "BEFORE_IMAGE", current.Value)
			if err != nil {
				return ledger.Release{}, wrap(CodeInternal, "Could not retain rollback material.", err)
			}
			plan.Operations[index].BeforeObjectHash = objectHash
		}
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return ledger.Release{}, err
	}
	intentID, err := ledger.NewID("intent")
	if err != nil {
		return ledger.Release{}, err
	}
	intent := ledger.ReleaseIntent{ID: intentID, LedgerID: ledgerID, ProposalID: proposal.ID, ProposalHash: proposal.Hash, ParentID: head.ReleaseID, Status: ledger.IntentReady, Plan: planBytes, CreatedAt: time.Now().UTC()}
	if err := engine.repository.SaveReleaseIntent(ctx, intent); err != nil {
		return ledger.Release{}, wrap(CodeInternal, "Could not persist Release Intent.", err)
	}
	if err := engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentApplying); err != nil {
		return ledger.Release{}, err
	}
	if err := engine.target.Apply(ctx, plan); err != nil {
		_ = engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentRecoveryRequired)
		return ledger.Release{}, wrap(CodeUnavailable, "Target apply failed; recovery is required.", err)
	}
	if err := engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentVerifying); err != nil {
		return ledger.Release{}, err
	}
	if err := engine.target.Verify(ctx, plan); err != nil {
		_ = engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentRecoveryRequired)
		return ledger.Release{}, wrap(CodeUnavailable, "Target verification failed; recovery is required.", err)
	}
	value, err := engine.newRelease(intent)
	if err != nil {
		return ledger.Release{}, err
	}
	if err := engine.repository.FinalizeRelease(ctx, intent, value); err != nil {
		return ledger.Release{}, wrap(CodeConflict, "Target applied, but HEAD finalization requires recovery.", err)
	}
	return value, nil
}
func (engine *Engine) newRelease(intent ledger.ReleaseIntent) (ledger.Release, error) {
	id, err := ledger.NewID("rel")
	if err != nil {
		return ledger.Release{}, err
	}
	value := ledger.Release{ID: id, LedgerID: intent.LedgerID, ProposalID: intent.ProposalID, ProposalHash: intent.ProposalHash, ParentID: intent.ParentID, CreatedAt: time.Now().UTC()}
	value.Hash, err = ledger.ReleaseHash(value)
	return value, err
}
func (engine *Engine) RecoverReleases(ctx context.Context) error {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	intents, err := engine.repository.ListUnfinishedReleaseIntents(ctx)
	if err != nil {
		return fmt.Errorf("list unfinished release intents: %w", err)
	}
	for _, intent := range intents {
		if intent.Status == ledger.IntentReady {
			continue
		}
		var plan targets.Plan
		if err := json.Unmarshal(intent.Plan, &plan); err != nil {
			_ = engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentRecoveryRequired)
			continue
		}
		if err := engine.target.Verify(ctx, plan); err != nil {
			_ = engine.repository.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentRecoveryRequired)
			continue
		}
		value, err := engine.newRelease(intent)
		if err != nil {
			return err
		}
		if err := engine.repository.FinalizeRelease(ctx, intent, value); err != nil {
			return fmt.Errorf("finalize recovered release %s: %w", intent.ID, err)
		}
	}
	return nil
}
