package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type rollbackState struct {
	action  ledger.ChangeAction
	desired json.RawMessage
}

// CreateRollbackProposal reconstructs the desired state at targetReleaseID and
// expresses it as ordinary Changes in a new forward-history Proposal.
func (engine *Engine) CreateRollbackProposal(ctx context.Context, ledgerID, targetReleaseID string) (ledger.Proposal, error) {
	releases, err := engine.repository.ListReleases(ctx, ledgerID)
	if err != nil {
		return ledger.Proposal{}, wrap(CodeInternal, "Could not load release history.", err)
	}
	targetIndex := -1
	for index, release := range releases {
		if release.ID == targetReleaseID {
			targetIndex = index
			break
		}
	}
	if targetIndex < 0 {
		return ledger.Proposal{}, wrap(CodeNotFound, "Rollback target Release was not found.", ledger.ErrInvalid)
	}
	if targetIndex == 0 {
		return ledger.Proposal{}, wrap(CodeConflict, "The selected Release is already HEAD.", ledger.ErrInvalid)
	}

	desiredByUnit := make(map[string]rollbackState)
	for _, release := range releases[:targetIndex] {
		intent, err := engine.repository.LoadReleaseIntentForProposal(ctx, release.ProposalID)
		if err != nil {
			return ledger.Proposal{}, wrap(CodeInternal, "Rollback material is unavailable.", err)
		}
		var plan targets.Plan
		if err := json.Unmarshal(intent.Plan, &plan); err != nil {
			return ledger.Proposal{}, wrap(CodeInternal, "Rollback plan is corrupt.", err)
		}
		for _, operation := range plan.Operations {
			state := rollbackState{action: ledger.ChangeDelete}
			if operation.BeforeExists {
				before, err := engine.repository.ReadObject(ctx, operation.BeforeObjectHash)
				if err != nil {
					return ledger.Proposal{}, wrap(CodeInternal, "Retained rollback value is unavailable.", err)
				}
				state.action = ledger.ChangePut
				state.desired = before
			}
			desiredByUnit[operation.Unit] = state
		}
	}

	units := make([]string, 0, len(desiredByUnit))
	for unit := range desiredByUnit {
		units = append(units, unit)
	}
	sort.Strings(units)
	changeIDs := make([]string, 0, len(units))
	for _, unit := range units {
		state := desiredByUnit[unit]
		change, err := engine.CreateChange(ctx, ledgerID, CreateChangeRequest{
			Unit:           unit,
			Action:         state.action,
			Desired:        state.desired,
			IdempotencyKey: fmt.Sprintf("rollback:%s:%s:%s", releases[0].ID, targetReleaseID, unit),
		})
		if err != nil {
			return ledger.Proposal{}, err
		}
		changeIDs = append(changeIDs, change.ID)
	}
	return engine.CreateProposal(ctx, ledgerID, CreateProposalRequest{Title: "Restore state from " + targetReleaseID, ChangeIDs: changeIDs})
}
