package engine

import (
	"context"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type SystemHealth struct {
	Database          string
	Target            string
	Inference         string
	InferenceState    string
	UnresolvedIntents int64
	PendingChanges    int64
	ObjectStoreBytes  int64
}

func (engine *Engine) Readiness(ctx context.Context) (bool, error) {
	return engine.repository.Readiness(ctx)
}

func (engine *Engine) ProbeHealth(ctx context.Context) SystemHealth {
	health := SystemHealth{Database: "ok", Target: "unknown", Inference: "disabled", InferenceState: engine.InferenceState()}
	stats, err := engine.repository.DatabaseStats(ctx)
	if err != nil {
		health.Database = "unreachable"
	} else {
		health.UnresolvedIntents = stats.UnresolvedIntents
		health.PendingChanges = stats.PendingChanges
		if bytes, sizeErr := engine.repository.ObjectStoreBytes(ctx); sizeErr == nil {
			health.ObjectStoreBytes = bytes
		}
	}
	if checker, ok := engine.targetHealth.(targets.HealthChecker); ok {
		if err := checker.Health(ctx); err != nil {
			health.Target = "unreachable"
			engine.metrics.TargetRequest("health", "error")
		} else {
			health.Target = "ok"
			engine.metrics.TargetRequest("health", "success")
		}
	}
	if engine.inference != nil {
		health.Inference = "unhealthy"
		if reporter, ok := engine.inference.(inference.StateReporter); ok {
			if reporter.Healthy() {
				health.Inference = "ok"
			}
		} else if checker, ok := engine.inference.(inference.HealthChecker); ok && checker.Health(ctx) == nil {
			health.Inference = "ok"
		}
	}
	return health
}
