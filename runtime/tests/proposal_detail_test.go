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
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

func newProposalDetailEngine(t *testing.T) (*engine.Engine, *repository.SQLite, string) {
	t.Helper()
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(context.Background(), filepath.Join(directory, "state.db"), filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	application := engine.New(repo, &memoryTarget{values: map[string]json.RawMessage{}}, nil)
	ledgerValue, err := application.CreateLedger(context.Background(), "Detail ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	return application, repo, ledgerValue.ID
}

func createProposalForDetail(t *testing.T, application *engine.Engine, ledgerID, suffix string) (ledger.Change, ledger.Proposal) {
	t.Helper()
	ctx := context.Background()
	change, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "unit-" + suffix, Action: ledger.ChangePut, Desired: json.RawMessage(`{"value":"` + suffix + `"}`), IdempotencyKey: "request-" + suffix})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Proposal " + suffix, ChangeIDs: []string{change.ID}})
	if err != nil {
		t.Fatal(err)
	}
	return change, proposal
}

func publicMessage(err error) string {
	_, message := engine.PublicError(err)
	return message
}

func TestProposalDetailGatesAndReleaseErrorsDoNotDrift(t *testing.T) {
	ctx := context.Background()
	application, _, ledgerID := newProposalDetailEngine(t)
	_, proposal := createProposalForDetail(t, application, ledgerID, "gates")

	detail, err := application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Gates.HasCurrentPassingCheck || detail.Gates.HasCurrentApproval || !detail.Gates.BaseMatchesHead || detail.Gates.Releasable {
		t.Fatalf("initial gates = %#v", detail.Gates)
	}
	if detail.Gates.ApprovalAction.Enabled || detail.Gates.ApprovalAction.Reason == "" || detail.Gates.ReleaseAction.Enabled || detail.Gates.ReleaseAction.Reason != detail.Gates.Reason {
		t.Fatalf("initial action gates = %#v", detail.Gates)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err == nil || publicMessage(err) != detail.Gates.ApprovalAction.Reason {
		t.Fatalf("approval error = %v, action reason = %q", err, detail.Gates.ApprovalAction.Reason)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err == nil || publicMessage(err) != detail.Gates.Reason {
		t.Fatalf("initial release error = %v, gate reason = %q", err, detail.Gates.Reason)
	}

	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	detail, err = application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Gates.HasCurrentPassingCheck || detail.Gates.HasCurrentApproval || detail.Gates.Releasable || !detail.Gates.ApprovalAction.Enabled || detail.Gates.ApprovalAction.Reason != "" || detail.Gates.ReleaseAction.Enabled || detail.Gates.ReleaseAction.Reason != detail.Gates.Reason || !strings.Contains(strings.ToLower(detail.Gates.Reason), "approval") {
		t.Fatalf("evaluated gates = %#v", detail.Gates)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerID, proposal.ID); err == nil || publicMessage(err) != detail.Gates.Reason {
		t.Fatalf("evaluated release error = %v, gate reason = %q", err, detail.Gates.Reason)
	}

	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	detail, err = application.LoadProposalDetail(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Gates.HasCurrentPassingCheck || !detail.Gates.HasCurrentApproval || !detail.Gates.BaseMatchesHead || !detail.Gates.Releasable || detail.Gates.Reason != "" || !detail.Gates.ApprovalAction.Enabled || !detail.Gates.ReleaseAction.Enabled || detail.Gates.ReleaseAction.Reason != "" {
		t.Fatalf("approved gates = %#v", detail.Gates)
	}
}

func TestProposalDetailDetectsMovedHead(t *testing.T) {
	ctx := context.Background()
	application, _, ledgerID := newProposalDetailEngine(t)
	_, first := createProposalForDetail(t, application, ledgerID, "first")
	_, stale := createProposalForDetail(t, application, ledgerID, "stale")
	for _, proposal := range []ledger.Proposal{first, stale} {
		if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
			t.Fatal(err)
		}
		if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := application.ReleaseProposal(ctx, ledgerID, first.ID); err != nil {
		t.Fatal(err)
	}
	detail, err := application.LoadProposalDetail(ctx, ledgerID, stale.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Gates.BaseMatchesHead || detail.Gates.Releasable || detail.CurrentHeadReleaseID == "" {
		t.Fatalf("stale gates = %#v, head = %q", detail.Gates, detail.CurrentHeadReleaseID)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerID, stale.ID); err == nil || publicMessage(err) != detail.Gates.Reason {
		t.Fatalf("stale release error = %v, gate reason = %q", err, detail.Gates.Reason)
	}
}

func TestProposalEvidenceApprovalsAndLedgerScope(t *testing.T) {
	ctx := context.Background()
	application, repo, ledgerID := newProposalDetailEngine(t)
	change, proposal := createProposalForDetail(t, application, ledgerID, "evidence")
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, proposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveCheckResult(ctx, ledger.CheckResult{ID: "chk_stale", ProposalID: proposal.ID, ProposalHash: "sha256:stale", Kind: "deterministic", Passed: true, Summary: "stale malformed", Evidence: []byte("not-json"), CreatedAt: time.Now().UTC().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveApproval(ctx, ledger.Approval{ID: "apr_stale", ProposalID: proposal.ID, ProposalHash: "sha256:stale", Actor: "old-reviewer", CreatedAt: time.Now().UTC().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}

	checks, err := application.ListCheckResults(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 || checks[0].ID != "chk_stale" || checks[0].Current || !checks[0].EvidenceUnavailable || checks[0].Evidence != nil || !checks[1].Current || !json.Valid(checks[1].Evidence) {
		t.Fatalf("checks = %#v", checks)
	}
	approvals, err := application.ListApprovals(ctx, ledgerID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 2 || approvals[0].ID != "apr_stale" || approvals[0].Current || !approvals[1].Current {
		t.Fatalf("approvals = %#v", approvals)
	}

	other, err := application.CreateLedger(ctx, "Other ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, load := range []func() error{
		func() error { _, err := application.LoadProposalDetail(ctx, other.ID, proposal.ID); return err },
		func() error { _, err := application.ListCheckResults(ctx, other.ID, proposal.ID); return err },
		func() error { _, err := application.ListApprovals(ctx, other.ID, proposal.ID); return err },
	} {
		if code, _ := engine.PublicError(load()); code != engine.CodeNotFound {
			t.Fatalf("cross-ledger code = %s", code)
		}
	}

	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)), "test")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerID+"/proposals/"+proposal.ID, nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "idempotencyKey") || strings.Contains(response.Body.String(), "requestFingerprint") || !strings.Contains(response.Body.String(), change.ID) {
		t.Fatalf("detail response %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerID+"/proposals/"+proposal.ID+"/checks", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"evidence":{"`) || !strings.Contains(response.Body.String(), `"evidenceUnavailable":true`) {
		t.Fatalf("checks response %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+ledgerID+"/proposals/"+proposal.ID+"/approvals", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "old-reviewer") {
		t.Fatalf("approvals response %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ledgers/"+other.ID+"/proposals/"+proposal.ID, nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"NOT_FOUND"`) {
		t.Fatalf("cross-ledger response %d: %s", response.Code, response.Body.String())
	}
}
