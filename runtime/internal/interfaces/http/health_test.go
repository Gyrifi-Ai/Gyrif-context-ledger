package httpinterface

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type healthTarget struct {
	healthErr     error
	healthStarted chan struct{}
	healthRelease chan struct{}
	healthCalls   atomic.Int32
}

func (target *healthTarget) Health(ctx context.Context) error {
	target.healthCalls.Add(1)
	if target.healthStarted != nil {
		select {
		case target.healthStarted <- struct{}{}:
		default:
		}
	}
	if target.healthRelease != nil {
		select {
		case <-target.healthRelease:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return target.healthErr
}
func (*healthTarget) Read(context.Context, string) (targets.Value, error) {
	return targets.Value{}, nil
}
func (*healthTarget) Fingerprint(context.Context, string) (string, error) { return "", nil }
func (*healthTarget) Preview(context.Context, []ledger.Change) (targets.Preview, error) {
	return targets.Preview{Fidelity: "FAST"}, nil
}
func (*healthTarget) Compile(context.Context, []ledger.Change) (targets.Plan, error) {
	return targets.Plan{}, nil
}
func (*healthTarget) Apply(context.Context, targets.Plan) error   { return nil }
func (*healthTarget) Verify(context.Context, targets.Plan) error  { return nil }
func (*healthTarget) Restore(context.Context, targets.Plan) error { return nil }
func (*healthTarget) Capabilities() targets.Capabilities          { return targets.Capabilities{} }

func testHealthServer(t *testing.T, target targets.TargetAdapter) (*Server, *engine.Engine, *repository.SQLite, string) {
	t.Helper()
	directory := t.TempDir()
	databasePath := filepath.Join(directory, "state.db")
	repo, err := repository.OpenSQLite(context.Background(), databasePath, filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	if target == nil {
		target = &healthTarget{}
	}
	application := engine.New(repo, target, nil)
	t.Cleanup(func() { _ = application.Close() })
	server := New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(server.Close)
	return server, application, repo, databasePath
}

func request(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestHealthzIsLockFreeAndNeverFallsThroughToStudio(t *testing.T) {
	server, application, _, _ := testHealthServer(t, nil)
	if err := application.Close(); err != nil {
		t.Fatal(err)
	}
	recorder := request(t, server.Handler(), http.MethodGet, "/healthz")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok" {
		t.Fatalf("healthz = %d %q", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q", contentType)
	}
	metrics := request(t, server.Handler(), http.MethodGet, "/metrics")
	if metrics.Code != http.StatusNotFound || !containsText(metrics.Body.String(), "NOT_FOUND") {
		t.Fatalf("main-listener metrics = %d %s", metrics.Code, metrics.Body.String())
	}
	other := request(t, server.MetricsHandler(), http.MethodGet, "/healthz")
	if other.Code != http.StatusNotFound || !containsText(other.Body.String(), "NOT_FOUND") {
		t.Fatalf("metrics-listener healthz = %d %s", other.Code, other.Body.String())
	}
}

func TestReadyzReportsDatabaseMigrationAndShutdownStates(t *testing.T) {
	t.Run("database unreachable", func(t *testing.T) {
		server, application, _, _ := testHealthServer(t, nil)
		if err := application.Close(); err != nil {
			t.Fatal(err)
		}
		recorder := request(t, server.Handler(), http.MethodGet, "/readyz")
		if recorder.Code != http.StatusServiceUnavailable || !json.Valid(recorder.Body.Bytes()) || !containsText(recorder.Body.String(), "database_unreachable") {
			t.Fatalf("readyz = %d %s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("migration incomplete", func(t *testing.T) {
		server, _, _, databasePath := testHealthServer(t, nil)
		db, err := sql.Open("sqlite", databasePath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version=(SELECT MAX(version) FROM schema_migrations)`); err != nil {
			t.Fatal(err)
		}
		recorder := request(t, server.Handler(), http.MethodGet, "/readyz")
		if recorder.Code != http.StatusServiceUnavailable || !containsText(recorder.Body.String(), "migrations_incomplete") {
			t.Fatalf("readyz = %d %s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("shutting down", func(t *testing.T) {
		server, application, _, _ := testHealthServer(t, nil)
		server.SetShuttingDown()
		if err := application.Close(); err != nil {
			t.Fatal(err)
		}
		recorder := request(t, server.Handler(), http.MethodGet, "/readyz")
		if recorder.Code != http.StatusServiceUnavailable || recorder.Body.String() != "{\"ready\":false,\"reasons\":[\"shutting_down\"]}\n" {
			t.Fatalf("readyz = %d %s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestReadyzIgnoresRecoveryRequiredIntents(t *testing.T) {
	server, application, repo, _ := testHealthServer(t, nil)
	ctx := context.Background()
	ledgerValue, err := application.CreateLedger(ctx, "ready", "")
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChange(ctx, ledgerValue.ID, engine.CreateChangeRequest{Unit: "unit", Action: ledger.ChangePut, Desired: json.RawMessage(`{"x":1}`), IdempotencyKey: "ready-1"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerValue.ID, engine.CreateProposalRequest{Title: "ready", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveReleaseIntent(ctx, ledger.ReleaseIntent{ID: "intent_ready", LedgerID: ledgerValue.ID, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Status: ledger.IntentRecoveryRequired, Plan: json.RawMessage(`{"operations":[]}`), CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	recorder := request(t, server.Handler(), http.MethodGet, "/readyz")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"ready\":true}\n" {
		t.Fatalf("readyz = %d %s", recorder.Code, recorder.Body.String())
	}
	health := application.ProbeHealth(ctx)
	server.health.value.Store(health)
	server.health.refreshed.Store(time.Now().UnixNano())
	status := request(t, server.Handler(), http.MethodGet, "/api/v1/system/status")
	if !containsText(status.Body.String(), `"unresolvedIntents":1`) {
		t.Fatalf("status = %s", status.Body.String())
	}
	metrics := request(t, server.MetricsHandler(), http.MethodGet, "/metrics")
	for _, expected := range []string{"gyrifi_unresolved_intents 1", "gyrifi_pending_changes 1", "gyrifi_object_store_bytes 7"} {
		if !containsText(metrics.Body.String(), expected) {
			t.Fatalf("metrics missing %q: %s", expected, metrics.Body.String())
		}
	}
}

func TestStatusUsesNonBlockingCachedDependencyHealth(t *testing.T) {
	target := &healthTarget{healthErr: errors.New("offline"), healthStarted: make(chan struct{}, 1), healthRelease: make(chan struct{})}
	server, _, _, _ := testHealthServer(t, target)
	started := time.Now()
	first := request(t, server.Handler(), http.MethodGet, "/api/v1/system/status")
	if first.Code != http.StatusOK || time.Since(started) > 100*time.Millisecond {
		t.Fatalf("first status took %s and returned %d", time.Since(started), first.Code)
	}
	select {
	case <-target.healthStarted:
	case <-time.After(time.Second):
		t.Fatal("target health probe did not start")
	}
	started = time.Now()
	second := request(t, server.Handler(), http.MethodGet, "/api/v1/system/status")
	if second.Code != http.StatusOK || time.Since(started) > 100*time.Millisecond {
		t.Fatalf("cached status took %s and returned %d", time.Since(started), second.Code)
	}
	if target.healthCalls.Load() != 1 {
		t.Fatalf("health calls = %d", target.healthCalls.Load())
	}
	close(target.healthRelease)
	deadline := time.Now().Add(time.Second)
	for server.health.value.Load().(engine.SystemHealth).Target != "unreachable" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if health := server.health.value.Load().(engine.SystemHealth); health.Target != "unreachable" || health.Database != "ok" || health.Inference != "disabled" {
		t.Fatalf("health = %+v", health)
	}
}

func containsText(value, expected string) bool {
	for index := 0; index+len(expected) <= len(value); index++ {
		if value[index:index+len(expected)] == expected {
			return true
		}
	}
	return false
}
