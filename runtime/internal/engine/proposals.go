package engine

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

type CreateProposalRequest struct {
	Title     string
	ChangeIDs []string
}

func (engine *Engine) CreateProposal(ctx context.Context, ledgerID string, request CreateProposalRequest) (ledger.Proposal, error) {
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" || len(request.ChangeIDs) == 0 {
		return ledger.Proposal{}, wrap(CodeInvalid, "Proposal title and at least one Change are required.", ledger.ErrInvalid)
	}
	changes, err := engine.repository.LoadChanges(ctx, ledgerID, request.ChangeIDs)
	if err != nil {
		return ledger.Proposal{}, wrap(CodeInvalid, "One or more selected Changes are unavailable.", err)
	}
	seen := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		if change.Status != ledger.ChangeReady {
			return ledger.Proposal{}, wrap(CodeConflict, "Only Ready Changes can be proposed.", ledger.ErrConflict)
		}
		if _, exists := seen[change.ID]; exists {
			return ledger.Proposal{}, wrap(CodeInvalid, "A Change may only be selected once.", ledger.ErrInvalid)
		}
		seen[change.ID] = struct{}{}
	}
	head, err := engine.repository.CurrentHead(ctx, ledgerID)
	if err != nil {
		return ledger.Proposal{}, wrap(CodeNotFound, "Ledger was not found.", err)
	}
	id, err := ledger.NewID("pr")
	if err != nil {
		return ledger.Proposal{}, wrap(CodeInternal, "Could not create Proposal.", err)
	}
	value := ledger.Proposal{ID: id, LedgerID: ledgerID, Title: request.Title, BaseReleaseID: head.ReleaseID, Status: ledger.ProposalDraft, ChangeIDs: append([]string(nil), request.ChangeIDs...), CreatedAt: time.Now().UTC()}
	value.Hash, err = ledger.ProposalHash(value)
	if err != nil {
		return ledger.Proposal{}, wrap(CodeInternal, "Could not hash Proposal.", err)
	}
	if err := engine.repository.InsertProposal(ctx, value); err != nil {
		return ledger.Proposal{}, wrap(CodeConflict, "One or more Changes are already in another active Proposal.", err)
	}
	engine.metrics.ProposalCreated()
	engine.publish(EventProposalCreated, ledgerID, value.ID)
	return value, nil
}

func (engine *Engine) CancelProposal(ctx context.Context, ledgerID, proposalID string) error {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	if err := engine.repository.CancelProposal(ctx, ledgerID, proposalID); err != nil {
		switch {
		case errors.Is(err, repository.ErrProposalAlreadyCancelled):
			return nil
		case errors.Is(err, repository.ErrNotFound):
			return wrap(CodeNotFound, "Proposal was not found.", err)
		case errors.Is(err, repository.ErrProposalReleased):
			return wrap(CodeConflict, "A released Proposal cannot be cancelled.", err)
		case errors.Is(err, repository.ErrProposalNotDraft):
			return wrap(CodeConflict, "Only a Draft Proposal can be cancelled.", err)
		case errors.Is(err, repository.ErrProposalReleaseStarted):
			return wrap(CodeConflict, "Release has already started for this Proposal.", err)
		default:
			return wrap(CodeInternal, "Could not cancel Proposal.", err)
		}
	}
	engine.publish(EventProposalCancelled, ledgerID, proposalID)
	return nil
}

func (engine *Engine) ApproveProposal(ctx context.Context, ledgerID, proposalID, actor string) error {
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return wrap(CodeNotFound, "Proposal was not found.", err)
	}
	gate, err := engine.evaluateApprovalGate(ctx, proposal)
	if err != nil {
		return wrap(CodeInternal, "Could not verify evaluation evidence.", err)
	}
	if !gate.Enabled {
		return wrap(CodeConflict, gate.Reason, ledger.ErrReleaseNotReady)
	}
	id, err := ledger.NewID("apr")
	if err != nil {
		return err
	}
	approval := ledger.Approval{ID: id, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Actor: strings.TrimSpace(actor), CreatedAt: time.Now().UTC()}
	if approval.Actor == "" {
		approval.Actor = "local-user"
	}
	if err := engine.repository.SaveApproval(ctx, approval); err != nil {
		if errors.Is(err, repository.ErrProposalAlreadyCancelled) {
			return wrap(CodeConflict, cancelledApproval, err)
		}
		return wrap(CodeInternal, "Could not save approval.", err)
	}
	engine.publish(EventProposalApproved, ledgerID, proposal.ID)
	return nil
}
