package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

func TestLedgerArchiveLifecycle(t *testing.T) {
	ctx := context.Background()
	application, _, ledgerID := newProposalDetailEngine(t)
	change, proposal := createProposalForDetail(t, application, ledgerID, "archive")
	response := lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/archive", nil)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "work in flight") {
		t.Fatalf("archive with draft = %d %s", response.Code, response.Body.String())
	}
	if err := application.CancelProposal(ctx, ledgerID, proposal.ID); err != nil {
		t.Fatal(err)
	}
	releasedChange, releasedProposal := createProposalForDetail(t, application, ledgerID, "released-history")
	if _, err := application.EvaluateProposal(ctx, ledgerID, releasedProposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerID, releasedProposal.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	release, err := application.ReleaseProposal(ctx, ledgerID, releasedProposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := application.Events().Subscribe(4)
	defer unsubscribe()
	response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/archive", nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("archive = %d %s", response.Code, response.Body.String())
	}
	if event := nextEvent(t, events); event.Kind != engine.EventLedgerArchived || event.LedgerID != ledgerID {
		t.Fatalf("archive event = %#v", event)
	}
	if response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/archive", nil); response.Code != http.StatusNoContent {
		t.Fatalf("second archive = %d %s", response.Code, response.Body.String())
	}
	select {
	case event := <-events:
		t.Fatalf("idempotent archive published %#v", event)
	default:
	}

	page, err := application.ListLedgers(ctx, engine.ListRequest{})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("default ledgers = %#v, %v", page.Items, err)
	}
	page, err = application.ListLedgers(ctx, engine.ListRequest{IncludeArchived: true})
	if err != nil || len(page.Items) != 1 || page.Items[0].ArchivedAt == nil {
		t.Fatalf("archived ledgers = %#v, %v", page.Items, err)
	}
	changes, err := application.ListChanges(ctx, ledgerID, engine.ListRequest{})
	if err != nil || len(changes.Items) != 2 || changes.Items[0].ID != releasedChange.ID || changes.Items[1].ID != change.ID {
		t.Fatalf("archived Change history = %#v, %v", changes.Items, err)
	}
	proposals, err := application.ListProposals(ctx, ledgerID, engine.ListRequest{})
	if err != nil || len(proposals.Items) != 2 || proposals.Items[0].ID != releasedProposal.ID || proposals.Items[1].ID != proposal.ID {
		t.Fatalf("archived Proposal history = %#v, %v", proposals.Items, err)
	}
	releases, err := application.ListReleases(ctx, ledgerID, engine.ListRequest{})
	if err != nil || len(releases.Items) != 1 || releases.Items[0].ID != release.ID {
		t.Fatalf("archived Release history = %#v, %v", releases.Items, err)
	}
	if _, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "blocked", Action: ledger.ChangeDelete, IdempotencyKey: "blocked"}); err == nil || err.Error() != "This Ledger is archived." {
		t.Fatalf("archived CreateChange error = %v", err)
	}
	if _, err := application.CreateProposal(ctx, ledgerID, engine.CreateProposalRequest{Title: "Blocked", ChangeIDs: []string{change.ID}}); err == nil || err.Error() != "This Ledger is archived." {
		t.Fatalf("archived CreateProposal error = %v", err)
	}
	if _, err := application.CreateRollbackProposal(ctx, ledgerID, "rel_unknown"); err == nil || err.Error() != "This Ledger is archived." {
		t.Fatalf("archived rollback error = %v", err)
	}

	response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/unarchive", nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unarchive = %d %s", response.Code, response.Body.String())
	}
	if event := nextEvent(t, events); event.Kind != engine.EventLedgerUnarchived {
		t.Fatalf("unarchive event = %#v", event)
	}
	if _, err := application.CreateChange(ctx, ledgerID, engine.CreateChangeRequest{Unit: "restored", Action: ledger.ChangeDelete, IdempotencyKey: "restored"}); err != nil {
		t.Fatalf("CreateChange after unarchive = %v", err)
	}
	if response = lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/unarchive", nil); response.Code != http.StatusNoContent {
		t.Fatalf("second unarchive = %d %s", response.Code, response.Body.String())
	}
}

func TestArchiveRejectsUnfinalizedReleaseIntent(t *testing.T) {
	ctx := context.Background()
	application, repo, ledgerID := newProposalDetailEngine(t)
	_, proposal := createProposalForDetail(t, application, ledgerID, "intent")
	if _, err := application.EvaluateProposal(ctx, ledgerID, proposal.ID, "safe"); err != nil {
		t.Fatal(err)
	}
	intent := ledger.ReleaseIntent{
		ID: "intent_archive_guard", LedgerID: ledgerID, ProposalID: proposal.ID, ProposalHash: proposal.Hash,
		Status: ledger.IntentRecoveryRequired, Plan: []byte(`{"operations":[]}`), CreatedAt: time.Now().UTC(),
	}
	if err := repo.SaveReleaseIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	response := lifecycleRequest(t, application, http.MethodPost, "/api/v1/ledgers/"+ledgerID+"/archive", nil)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "work in flight") {
		t.Fatalf("archive with unfinished intent = %d %s", response.Code, response.Body.String())
	}
}

func TestIncludeArchivedValidationAndWireMetadata(t *testing.T) {
	application, _, ledgerID := newProposalDetailEngine(t)
	if err := application.ArchiveLedger(context.Background(), ledgerID); err != nil {
		t.Fatal(err)
	}
	response := lifecycleRequest(t, application, http.MethodGet, "/api/v1/ledgers?includeArchived=true", nil)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"archivedAt"`) || !strings.Contains(response.Body.String(), ledgerID) {
		t.Fatalf("include archived = %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Items []ledger.Ledger `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || len(decoded.Items) != 1 || decoded.Items[0].ArchivedAt == nil {
		t.Fatalf("decoded archived ledger = %#v, %v", decoded, err)
	}
	response = lifecycleRequest(t, application, http.MethodGet, "/api/v1/ledgers?includeArchived=maybe", nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid includeArchived = %d %s", response.Code, response.Body.String())
	}
}
