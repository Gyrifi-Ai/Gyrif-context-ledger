package engine

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

type EvaluationResponse struct {
	Passed          bool                `json:"passed"`
	Summary         string              `json:"summary"`
	PreviewFidelity string              `json:"previewFidelity"`
	Findings        []inference.Finding `json:"findings,omitempty"`
}

func (engine *Engine) EvaluateProposal(ctx context.Context, ledgerID, proposalID, criteria string) (EvaluationResponse, error) {
	proposal, err := engine.repository.LoadProposal(ctx, ledgerID, proposalID)
	if err != nil {
		return EvaluationResponse{}, wrap(CodeNotFound, "Proposal was not found.", err)
	}
	if proposal.Status == ledger.ProposalCancelled {
		return EvaluationResponse{}, wrap(CodeConflict, "A cancelled Proposal cannot be evaluated.", ledger.ErrConflict)
	}
	changes, err := engine.repository.LoadChanges(ctx, ledgerID, proposal.ChangeIDs)
	if err != nil {
		return EvaluationResponse{}, wrap(CodeInternal, "Could not load proposed state.", err)
	}
	preview, err := engine.target.Preview(ctx, changes)
	if err != nil {
		return EvaluationResponse{}, wrap(CodeUnavailable, "Target preview is unavailable.", err)
	}
	contextValue, err := json.Marshal(struct {
		Proposal ledger.Proposal `json:"proposal"`
		Changes  []ledger.Change `json:"changes"`
		Preview  any             `json:"preview"`
	}{proposal, changes, preview})
	if err != nil {
		return EvaluationResponse{}, wrap(CodeInternal, "Could not construct effective proposed state.", err)
	}
	response := EvaluationResponse{Passed: true, Summary: "Deterministic proposal checks passed; local natural-language evaluation is disabled.", PreviewFidelity: preview.Fidelity}
	kind := "deterministic"
	var evidence []byte
	if engine.inference != nil {
		if !engine.InferenceReady() {
			return EvaluationResponse{}, wrap(CodeUnavailable, "Evaluation is unavailable: the inference process is not running.", errors.New("inference process is not ready"))
		}
		result, err := engine.inference.Evaluate(ctx, inference.EvaluationRequest{ProposalHash: proposal.Hash, Context: contextValue, Criteria: criteria})
		if err != nil {
			return EvaluationResponse{}, wrap(CodeUnavailable, "Local natural-language evaluation is unavailable.", err)
		}
		response.Passed = result.Passed
		response.Summary = result.Summary
		response.Findings = result.Findings
		kind = "natural-language"
		evidence, _ = json.Marshal(result)
	} else {
		evidence = contextValue
	}
	id, err := ledger.NewID("chk")
	if err != nil {
		return EvaluationResponse{}, err
	}
	check := ledger.CheckResult{ID: id, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Kind: kind, Passed: response.Passed, Summary: response.Summary, Evidence: evidence, CreatedAt: time.Now().UTC()}
	if err := engine.repository.SaveCheckResult(ctx, check); err != nil {
		if errors.Is(err, repository.ErrProposalAlreadyCancelled) {
			return EvaluationResponse{}, wrap(CodeConflict, "A cancelled Proposal cannot be evaluated.", err)
		}
		return EvaluationResponse{}, wrap(CodeInternal, "Could not persist evaluation evidence.", err)
	}
	engine.metrics.EvaluationCompleted(response.Passed)
	engine.publish(EventProposalEvaluated, ledgerID, proposal.ID)
	return response, nil
}
