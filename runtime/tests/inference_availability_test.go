package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

type unavailableInference struct{}

func (unavailableInference) Name() string  { return "llamacpp" }
func (unavailableInference) Healthy() bool { return false }
func (unavailableInference) State() string { return "restarting" }
func (unavailableInference) Evaluate(context.Context, inference.EvaluationRequest) (inference.EvaluationResult, error) {
	panic("Evaluate must not be called while inference is unavailable")
}

func TestUnavailableInferenceDoesNotPersistCheck(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(ctx, filepath.Join(directory, "state.db"), filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	application := engine.New(repo, &memoryTarget{values: map[string]json.RawMessage{}}, unavailableInference{})
	t.Cleanup(func() { _ = application.Close() })
	ledgerValue, err := application.CreateLedger(ctx, "Inference ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	_, proposal := createProposalForDetail(t, application, ledgerValue.ID, "inference-down")

	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(server.Close)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ledgers/"+ledgerValue.ID+"/proposals/"+proposal.ID+"/evaluation", strings.NewReader(`{"criteria":"must be safe"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "Evaluation is unavailable: the inference process is not running.") {
		t.Fatalf("evaluation response = %d %s", response.Code, response.Body.String())
	}
	statusRequest := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	statusResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK || !strings.Contains(statusResponse.Body.String(), `"inference":"unhealthy"`) || !strings.Contains(statusResponse.Body.String(), `"inferenceState":"restarting"`) {
		t.Fatalf("status response = %d %s", statusResponse.Code, statusResponse.Body.String())
	}
	checks, err := application.ListCheckResults(ctx, ledgerValue.ID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 0 {
		t.Fatalf("infrastructure outage persisted checks: %#v", checks)
	}
	if state := application.InferenceState(); state != "restarting" {
		t.Fatalf("inference state = %q", state)
	}
}
