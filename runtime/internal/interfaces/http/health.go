package httpinterface

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
)

type healthCache struct {
	engine     *engine.Engine
	value      atomic.Value
	refreshed  atomic.Int64
	refreshing atomic.Bool
	interval   time.Duration
	timeout    time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wait       sync.WaitGroup
}

func newHealthCache(application *engine.Engine) *healthCache {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &healthCache{engine: application, interval: 15 * time.Second, timeout: 5 * time.Second, ctx: ctx, cancel: cancel}
	inferenceHealth := "unhealthy"
	if application.InferenceName() == "disabled" {
		inferenceHealth = "disabled"
	}
	cache.value.Store(engine.SystemHealth{Database: "ok", Target: "unknown", Inference: inferenceHealth})
	return cache
}

func (cache *healthCache) current() engine.SystemHealth {
	last := time.Unix(0, cache.refreshed.Load())
	if time.Since(last) >= cache.interval && cache.refreshing.CompareAndSwap(false, true) {
		cache.wait.Add(1)
		go func() {
			defer cache.wait.Done()
			defer cache.refreshing.Store(false)
			ctx, cancel := context.WithTimeout(cache.ctx, cache.timeout)
			defer cancel()
			cache.value.Store(cache.engine.ProbeHealth(ctx))
			cache.refreshed.Store(time.Now().UnixNano())
		}()
	}
	return cache.value.Load().(engine.SystemHealth)
}

func (cache *healthCache) close() {
	cache.cancel()
	cache.wait.Wait()
}

func (server *Server) healthz(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

func (server *Server) readyz(writer http.ResponseWriter, request *http.Request) {
	if server.shuttingDown.Load() {
		server.writeJSON(writer, http.StatusServiceUnavailable, map[string]any{"ready": false, "reasons": []string{"shutting_down"}})
		return
	}
	reasons := make([]string, 0, 1)
	ctx, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	complete, err := server.engine.Readiness(ctx)
	if err != nil {
		reasons = append(reasons, "database_unreachable")
	} else if !complete {
		reasons = append(reasons, "migrations_incomplete")
	}
	if len(reasons) != 0 {
		server.writeJSON(writer, http.StatusServiceUnavailable, map[string]any{"ready": false, "reasons": reasons})
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"ready": true})
}

func (server *Server) SetShuttingDown() {
	server.shuttingDown.Store(true)
}
