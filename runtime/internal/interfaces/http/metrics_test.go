package httpinterface

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type metricsTarget struct {
	mutex    sync.Mutex
	values   map[string]targets.Value
	applyErr error
}

func (*metricsTarget) Health(context.Context) error { return nil }
func (target *metricsTarget) Read(_ context.Context, unit string) (targets.Value, error) {
	target.mutex.Lock()
	defer target.mutex.Unlock()
	if value, ok := target.values[unit]; ok {
		return value, nil
	}
	return targets.Value{Unit: unit}, nil
}
func (target *metricsTarget) Fingerprint(ctx context.Context, unit string) (string, error) {
	value, err := target.Read(ctx, unit)
	return value.Fingerprint, err
}
func (*metricsTarget) Preview(context.Context, []ledger.Change) (targets.Preview, error) {
	return targets.Preview{Fidelity: "FAST"}, nil
}
func (*metricsTarget) Compile(_ context.Context, changes []ledger.Change) (targets.Plan, error) {
	plan := targets.Plan{Operations: make([]targets.Operation, 0, len(changes))}
	for _, change := range changes {
		plan.Operations = append(plan.Operations, targets.Operation{Unit: change.Unit, Action: change.Action, Desired: change.Desired, DesiredFingerprint: change.DesiredFingerprint})
	}
	return plan, nil
}
func (target *metricsTarget) Apply(_ context.Context, plan targets.Plan) error {
	if target.applyErr != nil {
		return target.applyErr
	}
	target.mutex.Lock()
	defer target.mutex.Unlock()
	for _, operation := range plan.Operations {
		if operation.Action == ledger.ChangeDelete {
			delete(target.values, operation.Unit)
			continue
		}
		target.values[operation.Unit] = targets.Value{Unit: operation.Unit, Value: operation.Desired, Fingerprint: operation.DesiredFingerprint, Exists: true}
	}
	return nil
}
func (target *metricsTarget) Verify(ctx context.Context, plan targets.Plan) error {
	for _, operation := range plan.Operations {
		value, err := target.Read(ctx, operation.Unit)
		if err != nil {
			return err
		}
		if operation.Action == ledger.ChangeDelete && value.Exists {
			return &targets.VerificationError{}
		}
		if operation.Action == ledger.ChangePut && (!value.Exists || value.Fingerprint != operation.DesiredFingerprint) {
			return &targets.VerificationError{}
		}
	}
	return nil
}
func (target *metricsTarget) Restore(ctx context.Context, plan targets.Plan) error {
	return target.Apply(ctx, plan)
}
func (*metricsTarget) Capabilities() targets.Capabilities { return targets.Capabilities{} }

func metricsFlowServer(t *testing.T) (*Server, *engine.Engine, *Metrics, *metricsTarget) {
	t.Helper()
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(context.Background(), directory+"/state.db", directory+"/objects")
	if err != nil {
		t.Fatal(err)
	}
	metrics := NewMetrics()
	target := &metricsTarget{values: make(map[string]targets.Value)}
	application := engine.New(repo, target, nil, metrics)
	t.Cleanup(func() { _ = application.Close() })
	server := New(application, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics)
	t.Cleanup(server.Close)
	return server, application, metrics, target
}

func TestMetricsOutputHasPrometheusHeadersAndValidSamples(t *testing.T) {
	server, _, metrics, _ := metricsFlowServer(t)
	metrics.ChangeAccepted()
	metrics.TargetRequest("read", "success")
	response := request(t, server.MetricsHandler(), http.MethodGet, "/metrics")
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("metrics = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	body := response.Body.String()
	for _, expected := range []string{
		"# HELP gyrifi_http_requests_total",
		"# TYPE gyrifi_http_requests_total counter",
		"# TYPE gyrifi_http_request_duration_seconds histogram",
		"# TYPE gyrifi_unresolved_intents gauge",
		"gyrifi_changes_accepted_total 1",
		"gyrifi_target_requests_total{operation=\"read\",outcome=\"success\"} 1",
		"gyrifi_build_info{version=\"dev\",commit=\"unknown\"} 1",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, body)
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		separator := strings.LastIndexByte(line, ' ')
		if separator < 1 {
			t.Fatalf("invalid sample %q", line)
		}
		if _, err := strconv.ParseFloat(line[separator+1:], 64); err != nil {
			t.Fatalf("invalid sample value in %q: %v", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if escaped := metricEscape("a\\b\"c\nd"); escaped != `a\\b\"c\nd` {
		t.Fatalf("escaped label = %q", escaped)
	}
}

func TestMetricsUseRoutePatternsAndBoundedLabels(t *testing.T) {
	server, _, _, _ := metricsFlowServer(t)
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/healthz"}, {http.MethodGet, "/readyz"}, {http.MethodGet, "/api/v1/system/status"}, {http.MethodGet, "/api/v1/adapters"},
		{http.MethodGet, "/api/v1/ledgers"}, {http.MethodPost, "/api/v1/ledgers"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/changes"}, {http.MethodPost, "/api/v1/ledgers/ldg_secret123/changes"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/proposals"}, {http.MethodPost, "/api/v1/ledgers/ldg_secret123/proposals"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/checks"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/approvals"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/evaluation"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/approvals"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/cancel"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/proposals/pr_secret123/release"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/release-intents"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/release-intents/intent_secret123"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/release-intents/intent_secret123/retry"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/release-intents/intent_secret123/resolve"},
		{http.MethodGet, "/api/v1/ledgers/ldg_secret123/releases"},
		{http.MethodPost, "/api/v1/ledgers/ldg_secret123/releases/rel_secret123/rollback"},
	}
	for _, route := range routes {
		request(t, server.Handler(), route.method, route.path)
	}
	requestValue := httptest.NewRequest(http.MethodGet, "/events/v1?ledgerId=ldg_secret123", nil)
	ctx, cancel := context.WithCancel(requestValue.Context())
	cancel()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, requestValue.WithContext(ctx))

	response := request(t, server.MetricsHandler(), http.MethodGet, "/metrics")
	body := response.Body.String()
	if strings.Contains(body, "ldg_secret123") {
		t.Fatalf("metrics leaked a resource id:\n%s", body)
	}
	if !strings.Contains(body, `path_template="/api/v1/ledgers/{ledgerID}/changes"`) {
		t.Fatalf("metrics did not use route pattern:\n%s", body)
	}
	if strings.Contains(body, `path_template="unmatched"`) {
		t.Fatalf("registered route was emitted as unmatched:\n%s", body)
	}
}

func TestDomainCountersTrackDurableGovernanceFlow(t *testing.T) {
	server, application, _, _ := metricsFlowServer(t)
	ctx := context.Background()
	ledgerValue, err := application.CreateLedger(ctx, "metrics", "")
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChange(ctx, ledgerValue.ID, engine.CreateChangeRequest{Unit: "unit", Action: ledger.ChangeDelete, IdempotencyKey: "metric-change"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerValue.ID, engine.CreateProposalRequest{Title: "metrics", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerValue.ID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerValue.ID, proposal.ID, "operator"); err != nil {
		t.Fatal(err)
	}
	firstRelease, err := application.ReleaseProposal(ctx, ledgerValue.ID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondChange, err := application.CreateChange(ctx, ledgerValue.ID, engine.CreateChangeRequest{Unit: "unit", Action: ledger.ChangeDelete, IdempotencyKey: "metric-change-2"})
	if err != nil {
		t.Fatal(err)
	}
	secondProposal, err := application.CreateProposal(ctx, ledgerValue.ID, engine.CreateProposalRequest{Title: "metrics 2", ChangeIDs: []string{secondChange.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerValue.ID, secondProposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerValue.ID, secondProposal.ID, "operator"); err != nil {
		t.Fatal(err)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerValue.ID, secondProposal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := application.CreateRollbackProposal(ctx, ledgerValue.ID, firstRelease.ID); err != nil {
		t.Fatal(err)
	}
	body := request(t, server.MetricsHandler(), http.MethodGet, "/metrics").Body.String()
	for _, expected := range []string{
		"gyrifi_changes_accepted_total 3",
		"gyrifi_proposals_created_total 3",
		"gyrifi_evaluations_total{passed=\"true\"} 2",
		"gyrifi_releases_total{outcome=\"success\"} 2",
		"gyrifi_rollbacks_total 1",
		"gyrifi_target_requests_total{operation=\"apply\",outcome=\"success\"} 2",
		"gyrifi_target_requests_total{operation=\"verify\",outcome=\"success\"} 2",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, body)
		}
	}
}

func TestReleaseFailureCountersTrackApplyErrors(t *testing.T) {
	server, application, _, target := metricsFlowServer(t)
	ctx := context.Background()
	ledgerValue, err := application.CreateLedger(ctx, "failure metrics", "")
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChange(ctx, ledgerValue.ID, engine.CreateChangeRequest{Unit: "unit", Action: ledger.ChangeDelete, IdempotencyKey: "failure-metric-change"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerValue.ID, engine.CreateProposalRequest{Title: "failure", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerValue.ID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerValue.ID, proposal.ID, "operator"); err != nil {
		t.Fatal(err)
	}
	target.applyErr = errors.New("target offline")
	if _, err := application.ReleaseProposal(ctx, ledgerValue.ID, proposal.ID); err == nil {
		t.Fatal("expected release failure")
	}
	body := request(t, server.MetricsHandler(), http.MethodGet, "/metrics").Body.String()
	for _, expected := range []string{
		"gyrifi_releases_total{outcome=\"failure\"} 1",
		"gyrifi_target_requests_total{operation=\"apply\",outcome=\"error\"} 1",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, body)
		}
	}
}

func TestConcurrentHTTPMetricsAreRaceSafe(t *testing.T) {
	server, _, _, _ := metricsFlowServer(t)
	const count = 100
	var wait sync.WaitGroup
	wait.Add(count)
	for index := 0; index < count; index++ {
		go func() {
			defer wait.Done()
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		}()
	}
	wait.Wait()
	body := request(t, server.MetricsHandler(), http.MethodGet, "/metrics").Body.String()
	if !strings.Contains(body, "gyrifi_http_requests_total{method=\"GET\",path_template=\"/healthz\",status=\"200\"} 100") {
		t.Fatalf("concurrent request count missing:\n%s", body)
	}
}
