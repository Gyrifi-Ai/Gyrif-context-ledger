package inference

import (
	"context"
	"encoding/json"
)

type EvaluationRequest struct {
	ProposalHash string          `json:"proposalHash"`
	Context      json.RawMessage `json:"context"`
	Criteria     string          `json:"criteria"`
}

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Unit     string `json:"unit,omitempty"`
}

type EvaluationResult struct {
	Passed   bool      `json:"passed"`
	Summary  string    `json:"summary"`
	Findings []Finding `json:"findings"`
	Model    string    `json:"model"`
	Evidence any       `json:"evidence,omitempty"`
}

type Provider interface {
	Evaluate(context.Context, EvaluationRequest) (EvaluationResult, error)
	Name() string
}

type HealthChecker interface {
	Health(context.Context) error
}
