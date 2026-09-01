package tests

import (
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
)

func cancelProposalRequest(t *testing.T, application *engine.Engine, ledgerID, proposalID string) *httptest.ResponseRecorder {
	t.Helper()
	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/proposals/"+proposalID+"/cancel", nil)
	server.Handler().ServeHTTP(response, request)
	return response
}

func TestCancelProposalReleasesClaimsAndRetainsAudit(t *testing.T) {
	ctx := context.Background()
	application, _, ledgerID := newProposalDetailEngine(t)
	first, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "unit-first", Action: ledger.ChangePut, Desired: json.RawMessage(`{"value":1}`), IdempotencyKey: "cancel-first"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "unit-second", Action: ledger.ChangePut, Desired: json.RawMessage(`{"value":2}`), IdempotencyKey: "cancel-second"})
	if err != nil {
		t.Fatal(err)
	}
	waitForChangeStatus(t, application, ledgerID, first.ID, ledger.ChangeReady)
	waitForChangeStatus(t, application, ledgerID, second.ID, ledger.ChangeReady)
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Cancel me", ChangeIDs: []string{first.ID, second.ID}})
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := application.Events().Subscribe(2)
	defer unsubscribe()
	response := cancelProposalRequest(t, application, ledgerID, proposal.ID)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("cancel response = %d %s", response.Code, response.Body.String())
	}
	select {
	case event := <-events:
		if event.Kind != engine.EventProposalCancelled || event.LedgerID != ledgerID || event.SubjectID != proposal.ID {
			t.Fatalf("cancellation event = %#v", event)
		}
	default:
		t.Fatal("cancellation event was not published")
	}

	detail, err := application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Proposal.Status != ledger.ProposalCancelled || strings.Join(detail.Proposal.ChangeIDs, ",") != strings.Join([]string{first.ID, second.ID}, ",") {
		t.Fatalf("cancelled Proposal = %#v", detail.Proposal)
	}
	if detail.Gates.CancelAction.Enabled || detail.Gates.CancelAction.Reason != "This Proposal is already cancelled." || detail.Gates.ReleaseAction.Enabled || detail.Gates.ReleaseAction.Reason != "A cancelled Proposal cannot be released." {
		t.Fatalf("cancelled gates = %#v", detail.Gates)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err == nil || publicMessage(err) != "A cancelled Proposal cannot be evaluated." {
		t.Fatalf("cancelled evaluation error = %v", err)
	}
	changes, err := application.ListChanges(ctx, ledgerID, engine.ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range changes.Items {
		if change.Status != ledger.ChangeReady {
			t.Fatalf("Change %s status = %s", change.ID, change.Status)
		}
	}
	replacement, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Replacement", ChangeIDs: []string{second.ID, first.ID}})
	if err != nil {
		t.Fatalf("re-propose cancelled Changes: %v", err)
	}
	if replacement.Hash == proposal.Hash {
		t.Fatal("reordered replacement Proposal retained the cancelled hash")
	}
	if replacementChecks, err := application.ListCheckResults(ctx, ledgerID, replacement.ID); err != nil || len(replacementChecks) != 0 {
		t.Fatalf("replacement checks = %#v, %v", replacementChecks, err)
	}
	if replacementApprovals, err := application.ListApprovals(ctx, ledgerID, replacement.ID); err != nil || len(replacementApprovals) != 0 {
		t.Fatalf("replacement approvals = %#v, %v", replacementApprovals, err)
	}
	select {
	case event := <-events:
		if event.Kind != engine.EventProposalCreated || event.SubjectID != replacement.ID {
			t.Fatalf("replacement event = %#v", event)
		}
	default:
		t.Fatal("replacement creation event was not published")
	}

	response = cancelProposalRequest(t, application, ledgerID, proposal.ID)
	if response.Code != http.StatusNoContent {
		t.Fatalf("idempotent cancel response = %d %s", response.Code, response.Body.String())
	}
	select {
	case event := <-events:
		t.Fatalf("idempotent cancellation published %#v", event)
	default:
	}
}

func TestCancelProposalGuardsReleaseHistoryAndLedgerScope(t *testing.T) {
	ctx := context.Background()
	t.Run("non-draft", func(t *testing.T) {
		application, _, ledgerID := newProposalDetailEngine(t)
		_, proposal := createProposalForDetail(t, application, ledgerID, "reviewed-cancel")
		if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
			t.Fatal(err)
		}
		response := cancelProposalRequest(t, application, ledgerID, proposal.ID)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"message":"Only a Draft Proposal can be cancelled."`) {
			t.Fatalf("non-Draft cancel response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("released", func(t *testing.T) {
		application, _, ledgerID := newProposalDetailEngine(t)
		_, proposal := createProposalForDetail(t, application, ledgerID, "released-cancel")
		if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
			t.Fatal(err)
		}
		if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
			t.Fatal(err)
		}
		if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err != nil {
			t.Fatal(err)
		}
		response := cancelProposalRequest(t, application, ledgerID, proposal.ID)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"message":"A released Proposal cannot be cancelled."`) {
			t.Fatalf("released cancel response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("release intent", func(t *testing.T) {
		application, repo, ledgerID := newProposalDetailEngine(t)
		_, proposal := createProposalForDetail(t, application, ledgerID, "intent-cancel")
		intent := ledger.ReleaseIntent{ID: "intent_cancel_guard", LedgerID: ledgerID, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Status: ledger.IntentReady, Plan: []byte(`{"operations":[]}`), CreatedAt: time.Now().UTC()}
		if err := repo.SaveReleaseIntent(ctx, intent); err != nil {
			t.Fatal(err)
		}
		detail, err := application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
		if err != nil {
			t.Fatal(err)
		}
		if detail.Gates.CancelAction.Enabled || detail.Gates.CancelAction.Reason != "Release has already started for this Proposal." {
			t.Fatalf("intent cancellation gate = %#v", detail.Gates.CancelAction)
		}
		response := cancelProposalRequest(t, application, ledgerID, proposal.ID)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"message":"Release has already started for this Proposal."`) {
			t.Fatalf("intent cancel response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("not found in ledger", func(t *testing.T) {
		application, _, ledgerID := newProposalDetailEngine(t)
		_, proposal := createProposalForDetail(t, application, ledgerID, "scope-cancel")
		other, err := application.CreateLedger(ctx, "Cancellation scope", "")
		if err != nil {
			t.Fatal(err)
		}
		response := cancelProposalRequest(t, application, other.ID, proposal.ID)
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"NOT_FOUND"`) {
			t.Fatalf("cross-ledger cancel response = %d %s", response.Code, response.Body.String())
		}
	})
}
