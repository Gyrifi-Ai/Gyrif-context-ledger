package engine

import (
	"context"
	"encoding/json"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

const (
	passingCheckRequired         = "A current passing evaluation is required."
	approvalPassingCheckRequired = "A current passing evaluation is required before approval."
	approvalRequired             = "A current approval is required."
	headMoved                    = "Ledger HEAD moved after this Proposal was created."
	cancelledApproval            = "A cancelled Proposal cannot be approved."
	cancelledRelease             = "A cancelled Proposal cannot be released."
	alreadyReleased              = "This Proposal has already been released."
	alreadyCancelled             = "This Proposal is already cancelled."
	onlyDraftCancellation        = "Only a Draft Proposal can be cancelled."
	releaseAlreadyStarted        = "Release has already started for this Proposal."
)

type ActionGate struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type ProposalGates struct {
	HasCurrentPassingCheck bool       `json:"hasCurrentPassingCheck"`
	HasCurrentApproval     bool       `json:"hasCurrentApproval"`
	BaseMatchesHead        bool       `json:"baseMatchesHead"`
	Releasable             bool       `json:"releasable"`
	Reason                 string     `json:"reason"`
	ApprovalAction         ActionGate `json:"approvalAction"`
	ReleaseAction          ActionGate `json:"releaseAction"`
	CancelAction           ActionGate `json:"cancelAction"`
}

type ProposalDetail struct {
	Proposal             ledger.Proposal `json:"proposal"`
	Changes              []ledger.Change `json:"changes"`
	CurrentHeadReleaseID string          `json:"currentHeadReleaseId"`
	Gates                ProposalGates   `json:"gates"`
}

type CheckResult struct {
	ID                  string          `json:"id"`
	ProposalHash        string          `json:"proposalHash"`
	Kind                string          `json:"kind"`
	Passed              bool            `json:"passed"`
	Summary             string          `json:"summary"`
	Evidence            json.RawMessage `json:"evidence,omitempty"`
	EvidenceUnavailable bool            `json:"evidenceUnavailable,omitempty"`
	CreatedAt           string          `json:"createdAt"`
	Current             bool            `json:"current"`
}

type Approval struct {
	ID           string `json:"id"`
	ProposalHash string `json:"proposalHash"`
	Actor        string `json:"actor"`
	CreatedAt    string `json:"createdAt"`
	Current      bool   `json:"current"`
}

func (engine *Engine) evaluateApprovalGate(ctx context.Context, proposal ledger.Proposal) (ActionGate, error) {
	if proposal.Status == ledger.ProposalCancelled {
		return ActionGate{Reason: cancelledApproval}, nil
	}
	if proposal.Status == ledger.ProposalReleased {
		return ActionGate{Reason: alreadyReleased}, nil
	}
	passing, err := engine.repository.HasPassingCheck(ctx, proposal.ID, proposal.Hash)
	if err != nil {
		return ActionGate{}, err
	}
	gate := ActionGate{Enabled: passing}
	if !gate.Enabled {
		gate.Reason = approvalPassingCheckRequired
	}
	return gate, nil
}

func (engine *Engine) evaluateGates(ctx context.Context, proposal ledger.Proposal) (ProposalGates, string, error) {
	approvalAction, err := engine.evaluateApprovalGate(ctx, proposal)
	if err != nil {
		return ProposalGates{}, "", wrap(CodeInternal, "Could not verify checks.", err)
	}
	approved, err := engine.repository.HasApproval(ctx, proposal.ID, proposal.Hash)
	if err != nil {
		return ProposalGates{}, "", wrap(CodeInternal, "Could not verify approvals.", err)
	}
	head, err := engine.repository.CurrentHead(ctx, proposal.LedgerID)
	if err != nil {
		return ProposalGates{}, "", wrap(CodeInternal, "Could not load HEAD.", err)
	}
	gates := ProposalGates{
		HasCurrentPassingCheck: approvalAction.Enabled,
		HasCurrentApproval:     approved,
		BaseMatchesHead:        head.ReleaseID == proposal.BaseReleaseID,
		ApprovalAction:         approvalAction,
	}
	hasReleaseIntent, err := engine.repository.HasReleaseIntent(ctx, proposal.ID)
	if err != nil {
		return ProposalGates{}, "", wrap(CodeInternal, "Could not inspect Proposal release state.", err)
	}
	switch {
	case proposal.Status == ledger.ProposalReleased:
		gates.CancelAction.Reason = "A released Proposal cannot be cancelled."
	case proposal.Status == ledger.ProposalCancelled:
		gates.CancelAction.Reason = alreadyCancelled
	case hasReleaseIntent:
		gates.CancelAction.Reason = releaseAlreadyStarted
	case proposal.Status != ledger.ProposalDraft:
		gates.CancelAction.Reason = onlyDraftCancellation
	default:
		gates.CancelAction.Enabled = true
	}
	gates.Releasable = gates.HasCurrentPassingCheck && gates.HasCurrentApproval && gates.BaseMatchesHead && proposal.Status != ledger.ProposalCancelled && proposal.Status != ledger.ProposalReleased
	switch {
	case proposal.Status == ledger.ProposalCancelled:
		gates.Reason = cancelledRelease
	case proposal.Status == ledger.ProposalReleased:
		gates.Reason = alreadyReleased
	case !gates.HasCurrentPassingCheck:
		gates.Reason = passingCheckRequired
	case !gates.HasCurrentApproval:
		gates.Reason = approvalRequired
	case !gates.BaseMatchesHead:
		gates.Reason = headMoved
	}
	gates.ReleaseAction = ActionGate{Enabled: gates.Releasable, Reason: gates.Reason}
	return gates, head.ReleaseID, nil
}

func (engine *Engine) LoadProposalDetail(ctx context.Context, ledgerID, proposalID string) (ProposalDetail, error) {
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return ProposalDetail{}, wrap(CodeNotFound, "Proposal was not found.", err)
	}
	changes, err := engine.repository.LoadChanges(ctx, ledgerID, proposal.ChangeIDs)
	if err != nil {
		return ProposalDetail{}, wrap(CodeInternal, "Could not load Proposal Changes.", err)
	}
	gates, headReleaseID, err := engine.evaluateGates(ctx, proposal)
	if err != nil {
		return ProposalDetail{}, err
	}
	return ProposalDetail{Proposal: proposal, Changes: changes, CurrentHeadReleaseID: headReleaseID, Gates: gates}, nil
}

func (engine *Engine) ListCheckResults(ctx context.Context, ledgerID, proposalID string) ([]CheckResult, error) {
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return nil, wrap(CodeNotFound, "Proposal was not found.", err)
	}
	stored, err := engine.repository.ListCheckResults(ctx, proposal.ID)
	if err != nil {
		return nil, wrap(CodeInternal, "Could not load evaluation evidence.", err)
	}
	items := make([]CheckResult, 0, len(stored))
	for _, value := range stored {
		item := CheckResult{ID: value.ID, ProposalHash: value.ProposalHash, Kind: value.Kind, Passed: value.Passed, Summary: value.Summary, CreatedAt: value.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), Current: value.ProposalHash == proposal.Hash}
		if json.Valid(value.Evidence) {
			item.Evidence = append(json.RawMessage(nil), value.Evidence...)
		} else {
			item.EvidenceUnavailable = true
		}
		items = append(items, item)
	}
	return items, nil
}

func (engine *Engine) ListApprovals(ctx context.Context, ledgerID, proposalID string) ([]Approval, error) {
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return nil, wrap(CodeNotFound, "Proposal was not found.", err)
	}
	stored, err := engine.repository.ListApprovals(ctx, proposal.ID)
	if err != nil {
		return nil, wrap(CodeInternal, "Could not load approvals.", err)
	}
	items := make([]Approval, 0, len(stored))
	for _, value := range stored {
		items = append(items, Approval{ID: value.ID, ProposalHash: value.ProposalHash, Actor: value.Actor, CreatedAt: value.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), Current: value.ProposalHash == proposal.Hash})
	}
	return items, nil
}
