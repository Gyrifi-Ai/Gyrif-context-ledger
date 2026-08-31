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
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

func approvedProposal(t *testing.T, application *engine.Engine, ledgerID, suffix string) ledger.Proposal {
	t.Helper()
	ctx := context.Background()
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{
		Unit:           "unit-" + suffix,
		Action:         ledger.ChangePut,
		Desired:        json.RawMessage(`{"id":"unit-` + suffix + `","payload":{"version":"desired"}}`),
		IdempotencyKey: "request-" + suffix,
	})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Proposal " + suffix, ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	return proposal
}

func recoveryIntent(t *testing.T, application *engine.Engine, ledgerID, suffix string) (ledger.Proposal, engine.ReleaseIntent) {
	t.Helper()
	proposal := approvedProposal(t, application, ledgerID, suffix)
	if _, err := application.ReleaseProposal(context.Background(), ledgerID, proposal.ID); err == nil {
		t.Fatal("release should enter recovery")
	}
	items, err := application.ListReleaseIntents(context.Background(), ledgerID, nil)
	if err != nil || len(items) == 0 {
		t.Fatalf("release intents = %#v, %v", items, err)
	}
	return proposal, items[0]
}

func errorCode(err error) engine.ErrorCode {
	code, _ := engine.PublicError(err)
	return code
}

func TestRetryReleaseIntentFinalizesWithoutReapplying(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}, failVerify: true}
	application, ledgerID := newEngine(t, target)
	_, intent := recoveryIntent(t, application, ledgerID, "retry")
	applyCalls := target.applyCalls
	events, unsubscribe := application.Events().Subscribe(2)
	defer unsubscribe()

	target.failVerify = false
	result, err := application.RetryReleaseIntent(ctx, ledgerID, intent.ID)
	if err != nil || !result.Resolved || len(result.Mismatches) != 0 {
		t.Fatalf("retry = %#v, %v", result, err)
	}
	if target.applyCalls != applyCalls {
		t.Fatalf("retry applied target: calls %d -> %d", applyCalls, target.applyCalls)
	}
	if event := nextEvent(t, events); event.Kind != engine.EventReleaseCompleted {
		t.Fatalf("retry completion event = %s", event.Kind)
	}
	if event := nextEvent(t, events); event.Kind != engine.EventIntentResolved || event.SubjectID != intent.ID {
		t.Fatalf("intent resolution event = %#v", event)
	}
	releases, err := application.ListReleases(ctx, ledgerID)
	if err != nil || len(releases) != 1 {
		t.Fatalf("releases = %#v, %v", releases, err)
	}
	loaded, err := application.LoadReleaseIntent(ctx, ledgerID, intent.ID)
	if err != nil || loaded.Status != ledger.IntentFinalized {
		t.Fatalf("intent = %#v, %v", loaded, err)
	}
	if _, err := application.RetryReleaseIntent(ctx, ledgerID, intent.ID); errorCode(err) != engine.CodeConflict || !strings.Contains(err.Error(), "FINALIZED") {
		t.Fatalf("second retry error = %v", err)
	}
}

func TestRetryReleaseIntentReportsMismatchesAndUnavailable(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}, failApply: true}
	application, ledgerID := newEngine(t, target)
	_, intent := recoveryIntent(t, application, ledgerID, "mismatch")

	result, err := application.RetryReleaseIntent(ctx, ledgerID, intent.ID)
	if err != nil || result.Resolved || len(result.Mismatches) != 1 || result.Mismatches[0].Unit != "unit-mismatch" || result.Mismatches[0].Expected == "" || result.Mismatches[0].Observed != "" {
		t.Fatalf("mismatch retry = %#v, %v", result, err)
	}
	target.failRead = true
	if _, err := application.RetryReleaseIntent(ctx, ledgerID, intent.ID); errorCode(err) != engine.CodeUnavailable {
		t.Fatalf("unavailable retry error = %v", err)
	}
	loaded, err := application.LoadReleaseIntent(ctx, ledgerID, intent.ID)
	if err != nil || loaded.Status != ledger.IntentRecoveryRequired {
		t.Fatalf("intent changed after unavailable retry: %#v, %v", loaded, err)
	}
}

func TestResolveReleaseIntentUnblocksReleaseWithoutAdvancingHead(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{}, failApply: true}
	application, ledgerID := newEngine(t, target)
	proposal, intent := recoveryIntent(t, application, ledgerID, "resolve")
	second := approvedProposal(t, application, ledgerID, "guard")

	if _, err := application.ReleaseProposal(ctx, ledgerID, second.ID); errorCode(err) != engine.CodeConflict || err.Error() != "A release intent requires recovery before further releases." {
		t.Fatalf("release guard error = %v", err)
	}
	if err := application.ResolveReleaseIntent(ctx, ledgerID, intent.ID, "OTHER", "operator checked"); errorCode(err) != engine.CodeInvalid {
		t.Fatalf("invalid resolution error = %v", err)
	}
	if err := application.ResolveReleaseIntent(ctx, ledgerID, intent.ID, string(ledger.IntentAbandoned), "  "); errorCode(err) != engine.CodeInvalid {
		t.Fatalf("empty note error = %v", err)
	}
	if err := application.ResolveReleaseIntent(ctx, ledgerID, intent.ID, string(ledger.IntentAbandoned), "Target repaired manually"); err != nil {
		t.Fatal(err)
	}
	loaded, err := application.LoadReleaseIntent(ctx, ledgerID, intent.ID)
	if err != nil || loaded.Status != ledger.IntentAbandoned || loaded.Resolution != string(ledger.IntentAbandoned) || loaded.ResolutionNote != "Target repaired manually" || loaded.ResolvedAt == nil {
		t.Fatalf("resolved intent = %#v, %v", loaded, err)
	}
	detail, err := application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
	if err != nil || detail.Proposal.Status != ledger.ProposalApproved || detail.CurrentHeadReleaseID != "" {
		t.Fatalf("proposal/head changed during resolve: %#v, %v", detail, err)
	}
	target.failApply = false
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err != nil {
		t.Fatalf("release after abandon = %v", err)
	}
}

func TestReleaseIntentReadAPIAndBeforeImagePresence(t *testing.T) {
	ctx := context.Background()
	target := &memoryTarget{values: map[string]json.RawMessage{"unit-read": json.RawMessage(`{"id":"unit-read","payload":{"version":"before"}}`)}, failVerify: true}
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(ctx, directory+"/state.db", directory+"/objects")
	if err != nil {
		t.Fatal(err)
	}
	application := engine.New(repo, target, nil)
	t.Cleanup(func() { _ = application.Close() })
	ledgerValue, err := application.CreateLedger(ctx, "Read ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	_, intent := recoveryIntent(t, application, ledgerValue.ID, "read")
	loaded, err := application.LoadReleaseIntent(ctx, ledgerValue.ID, intent.ID)
	if err != nil || len(loaded.Plan.Operations) != 1 || !loaded.Plan.Operations[0].HasBeforeImage {
		t.Fatalf("intent detail = %#v, %v", loaded, err)
	}

	missingPlan, _ := json.Marshal(targets.Plan{Operations: []targets.Operation{{Unit: "missing", Action: ledger.ChangePut, DesiredFingerprint: "sha256:desired", BeforeExists: true, BeforeObjectHash: "sha256:" + strings.Repeat("0", 64)}}})
	missing := ledger.ReleaseIntent{ID: "intent_missing_object", LedgerID: ledgerValue.ID, ProposalID: loaded.ProposalID, ProposalHash: loaded.ProposalHash, Status: ledger.IntentRecoveryRequired, Plan: missingPlan, CreatedAt: time.Now().UTC().Add(time.Minute)}
	if err := repo.SaveReleaseIntent(ctx, missing); err != nil {
		t.Fatal(err)
	}
	missingView, err := application.LoadReleaseIntent(ctx, ledgerValue.ID, missing.ID)
	if err != nil || missingView.Plan.Operations[0].HasBeforeImage {
		t.Fatalf("missing before-image = %#v, %v", missingView, err)
	}

	other, err := application.CreateLedger(ctx, "Other read ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.LoadReleaseIntent(ctx, other.ID, intent.ID); errorCode(err) != engine.CodeNotFound {
		t.Fatalf("cross-ledger load error = %v", err)
	}

	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents?status=RECOVERY_REQUIRED", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"hasBeforeImage":true`) || !strings.Contains(response.Body.String(), intent.ID) {
		t.Fatalf("intent list response %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents/"+intent.ID, nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"plan":{"operations"`) {
		t.Fatalf("intent detail response %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents?status=UNKNOWN", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_ARGUMENT"`) {
		t.Fatalf("invalid status response %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+other.ID+"/release-intents/"+intent.ID, nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-ledger response %d: %s", response.Code, response.Body.String())
	}
	if err := repo.UpdateReleaseIntent(ctx, intent.ID, ledger.IntentVerifying); err != nil {
		t.Fatal(err)
	}
	target.mu.Lock()
	target.failVerify = false
	target.values["unit-read"] = json.RawMessage(`{"id":"unit-read","payload":{"version":"foreign"}}`)
	target.mu.Unlock()
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents/"+intent.ID+"/retry", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"resolved":false`) || !strings.Contains(response.Body.String(), `"unit":"unit-read"`) {
		t.Fatalf("retry mismatch response %d: %s", response.Code, response.Body.String())
	}
	retried, err := application.LoadReleaseIntent(ctx, ledgerValue.ID, intent.ID)
	if err != nil || retried.Status != ledger.IntentRecoveryRequired {
		t.Fatalf("retry did not restore recovery status: %#v, %v", retried, err)
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents/"+intent.ID+"/resolve", bytes.NewBufferString(`{"resolution":"ABANDONED","note":"operator decision"}`))
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("resolve response %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/ledgers/"+ledgerValue.ID+"/release-intents/"+intent.ID+"/resolve", bytes.NewBufferString(`{"resolution":"ABANDONED","note":"again"}`))
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("second resolve response %d: %s", response.Code, response.Body.String())
	}
}
