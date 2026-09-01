package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

func lifecycleRequest(t *testing.T, application *engine.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(server.Close)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func TestChangeWithdrawalLifecycle(t *testing.T) {
	ctx := context.Background()
	application, ledgerID := newEngine(t, &memoryTarget{values: map[string]json.RawMessage{}})
	changeRequest := engine.CreateChangeRequest{Unit: "mistake", Action: ledger.ChangePut, Desired: json.RawMessage(`{"wrong":true}`), IdempotencyKey: "mistake-1"}
	change, err := application.CreateChange(ctx, ledgerID, changeRequest)
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := application.Events().Subscribe(2)
	defer unsubscribe()

	response := lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+change.ID+"/withdraw", []byte(`{"reason":"misconfigured source"}`))
	if response.Code != http.StatusNoContent {
		t.Fatalf("withdraw = %d %s", response.Code, response.Body.String())
	}
	if event := nextEvent(t, events); event.Kind != engine.EventChangeWithdrawn || event.SubjectID != change.ID {
		t.Fatalf("withdraw event = %#v", event)
	}
	page, err := application.ListChanges(ctx, ledgerID, engine.ListRequest{})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("default inbox = %#v, %v", page.Items, err)
	}
	withdrawn, err := application.ListChanges(ctx, ledgerID, engine.ListRequest{Status: string(ledger.ChangeWithdrawn)})
	if err != nil || len(withdrawn.Items) != 1 || withdrawn.Items[0].ID != change.ID {
		t.Fatalf("withdrawn filter = %#v, %v", withdrawn.Items, err)
	}
	duplicate, err := application.CreateChange(ctx, ledgerID, changeRequest)
	if err != nil || duplicate.ID != change.ID || duplicate.Status != ledger.ChangeWithdrawn {
		t.Fatalf("idempotent ingestion = %#v, %v", duplicate, err)
	}
	response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+change.ID+"/withdraw", []byte(`{"reason":"retry"}`))
	if response.Code != http.StatusNoContent {
		t.Fatalf("second withdraw = %d %s", response.Code, response.Body.String())
	}
	select {
	case event := <-events:
		t.Fatalf("idempotent withdrawal published %#v", event)
	default:
	}
	if _, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Bad selection", ChangeIDs: []string{change.ID}}); err == nil || !strings.Contains(err.Error(), "WITHDRAWN") {
		t.Fatalf("withdrawn Change proposal error = %v", err)
	}
}

func TestWithdrawalGuardsAndReasonValidation(t *testing.T) {
	ctx := context.Background()
	application, ledgerID := newEngine(t, &memoryTarget{values: map[string]json.RawMessage{}})
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "claimed", Action: ledger.ChangeDelete, IdempotencyKey: "claimed-1"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Claim", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	response := lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+change.ID+"/withdraw", []byte(`{"reason":"wrong"}`))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "Proposal "+proposal.ID) || !strings.Contains(response.Body.String(), "Cancel the Proposal first") {
		t.Fatalf("claimed withdrawal = %d %s", response.Code, response.Body.String())
	}
	if err := application.CancelProposal(ctx, ledgerID, proposal.ID); err != nil {
		t.Fatal(err)
	}
	if response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+change.ID+"/withdraw", []byte(`{"reason":"wrong"}`)); response.Code != http.StatusNoContent {
		t.Fatalf("withdraw after cancel = %d %s", response.Code, response.Body.String())
	}

	missing, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "missing-reason", Action: ledger.ChangeDelete, IdempotencyKey: "missing-reason"})
	if err != nil {
		t.Fatal(err)
	}
	response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+missing.ID+"/withdraw", []byte(`{"reason":"  "}`))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Withdrawal reason is required") {
		t.Fatalf("empty reason = %d %s", response.Code, response.Body.String())
	}
	response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+missing.ID+"/withdraw", []byte(`{}`))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Withdrawal reason is required") {
		t.Fatalf("missing reason = %d %s", response.Code, response.Body.String())
	}
}

func TestReleasedChangeCannotBeWithdrawn(t *testing.T) {
	ctx := context.Background()
	application, ledgerID := newEngine(t, &memoryTarget{values: map[string]json.RawMessage{}})
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "released", Action: ledger.ChangePut, Desired: json.RawMessage(`{"ok":true}`), IdempotencyKey: "released-1"})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Release", ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err = application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err = application.ReleaseProposal(ctx, ledgerID, proposal.ID); err != nil {
		t.Fatal(err)
	}
	response := lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/changes/"+change.ID+"/withdraw", []byte(`{"reason":"too late"}`))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "A released Change is part of the audit trail and cannot be withdrawn.") {
		t.Fatalf("released withdrawal = %d %s", response.Code, response.Body.String())
	}
}
