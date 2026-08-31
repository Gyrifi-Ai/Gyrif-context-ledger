package engine

import (
	"context"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type MetricSink interface {
	ChangeAccepted()
	ProposalCreated()
	EvaluationCompleted(bool)
	ReleaseCompleted(string)
	RollbackCreated()
	TargetRequest(operation, outcome string)
}

type discardMetrics struct{}

func (discardMetrics) ChangeAccepted()              {}
func (discardMetrics) ProposalCreated()             {}
func (discardMetrics) EvaluationCompleted(bool)     {}
func (discardMetrics) ReleaseCompleted(string)      {}
func (discardMetrics) RollbackCreated()             {}
func (discardMetrics) TargetRequest(string, string) {}

type meteredTarget struct {
	target  targets.TargetAdapter
	metrics MetricSink
}

func (target *meteredTarget) observe(operation string, err error) {
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	target.metrics.TargetRequest(operation, outcome)
}

func (target *meteredTarget) Read(ctx context.Context, unit string) (value targets.Value, err error) {
	defer func() { target.observe("read", err) }()
	return target.target.Read(ctx, unit)
}

func (target *meteredTarget) Fingerprint(ctx context.Context, unit string) (value string, err error) {
	defer func() { target.observe("fingerprint", err) }()
	return target.target.Fingerprint(ctx, unit)
}

func (target *meteredTarget) Preview(ctx context.Context, changes []ledger.Change) (value targets.Preview, err error) {
	defer func() { target.observe("preview", err) }()
	return target.target.Preview(ctx, changes)
}

func (target *meteredTarget) Compile(ctx context.Context, changes []ledger.Change) (value targets.Plan, err error) {
	defer func() { target.observe("compile", err) }()
	return target.target.Compile(ctx, changes)
}

func (target *meteredTarget) Apply(ctx context.Context, plan targets.Plan) (err error) {
	defer func() { target.observe("apply", err) }()
	return target.target.Apply(ctx, plan)
}

func (target *meteredTarget) Verify(ctx context.Context, plan targets.Plan) (err error) {
	defer func() { target.observe("verify", err) }()
	return target.target.Verify(ctx, plan)
}

func (target *meteredTarget) Restore(ctx context.Context, plan targets.Plan) (err error) {
	defer func() { target.observe("restore", err) }()
	return target.target.Restore(ctx, plan)
}

func (target *meteredTarget) Capabilities() targets.Capabilities {
	return target.target.Capabilities()
}
