package targets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

type Capabilities struct {
	AtomicApply      bool `json:"atomicApply"`
	ExactPreview     bool `json:"exactPreview"`
	ConditionalWrite bool `json:"conditionalWrite"`
	Batch            bool `json:"batch"`
	Restore          bool `json:"restore"`
}

type Value struct {
	Unit        string          `json:"unit"`
	Value       json.RawMessage `json:"value"`
	Fingerprint string          `json:"fingerprint"`
	Exists      bool            `json:"exists"`
}

type Preview struct {
	Fidelity string `json:"fidelity"`
	Summary  string `json:"summary"`
}

type Operation struct {
	Unit                string              `json:"unit"`
	Action              ledger.ChangeAction `json:"action"`
	Desired             json.RawMessage     `json:"desired,omitempty"`
	ExpectedFingerprint string              `json:"expectedFingerprint,omitempty"`
	ExpectedExists      bool                `json:"expectedExists"`
	DesiredFingerprint  string              `json:"desiredFingerprint"`
	TargetMetric        string              `json:"targetMetric,omitempty"`
	BeforeObjectHash    string              `json:"beforeObjectHash,omitempty"`
	BeforeExists        bool                `json:"beforeExists"`
}

type Plan struct {
	Operations []Operation `json:"operations"`
}

type VerificationMismatch struct {
	Unit     string `json:"unit"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type VerificationError struct {
	Mismatches []VerificationMismatch
}

func (err *VerificationError) Error() string {
	return fmt.Sprintf("target verification found %d mismatched units", len(err.Mismatches))
}

type TargetAdapter interface {
	Read(context.Context, string) (Value, error)
	Fingerprint(context.Context, string) (string, error)
	Preview(context.Context, []ledger.Change) (Preview, error)
	Compile(context.Context, []ledger.Change) (Plan, error)
	Apply(context.Context, Plan) error
	Verify(context.Context, Plan) error
	Restore(context.Context, Plan) error
	Capabilities() Capabilities
}
