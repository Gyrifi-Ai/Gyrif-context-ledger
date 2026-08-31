package repository

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

func proposalRepository(t *testing.T) (*SQLite, ledger.Proposal) {
	t.Helper()
	ctx := context.Background()
	directory := t.TempDir()
	repository, err := OpenSQLite(ctx, filepath.Join(directory, "state.db"), filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	ledgerValue := ledger.Ledger{ID: "ldg_test", Name: "Test", CreatedAt: time.Now().UTC()}
	if err := repository.CreateLedger(ctx, ledgerValue); err != nil {
		t.Fatal(err)
	}
	change := ledger.Change{ID: "chg_test", LedgerID: ledgerValue.ID, Unit: "unit", Action: ledger.ChangePut, Desired: json.RawMessage(`{"value":1}`), DesiredFingerprint: "sha256:desired", IdempotencyKey: "request", RequestFingerprint: "sha256:request", Status: ledger.ChangeReady, CreatedAt: time.Now().UTC()}
	if err := repository.InsertChange(ctx, &change); err != nil {
		t.Fatal(err)
	}
	proposal := ledger.Proposal{ID: "pr_test", LedgerID: ledgerValue.ID, Title: "Test", Hash: "sha256:current", Status: ledger.ProposalDraft, ChangeIDs: []string{change.ID}, CreatedAt: time.Now().UTC()}
	if err := repository.InsertProposal(ctx, proposal); err != nil {
		t.Fatal(err)
	}
	return repository, proposal
}

func TestListCheckResultsNewestFirst(t *testing.T) {
	ctx := context.Background()
	repository, proposal := proposalRepository(t)
	older := time.Date(2026, 8, 31, 7, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	for _, value := range []ledger.CheckResult{
		{ID: "chk_older", ProposalID: proposal.ID, ProposalHash: proposal.Hash, Kind: "deterministic", Passed: true, Summary: "older", Evidence: []byte(`{"order":1}`), CreatedAt: older},
		{ID: "chk_newer", ProposalID: proposal.ID, ProposalHash: "sha256:stale", Kind: "deterministic", Passed: false, Summary: "newer", Evidence: []byte(`{"order":2}`), CreatedAt: newer},
	} {
		if err := repository.SaveCheckResult(ctx, value); err != nil {
			t.Fatal(err)
		}
	}

	items, err := repository.ListCheckResults(ctx, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "chk_newer" || items[1].ID != "chk_older" {
		t.Fatalf("checks = %#v", items)
	}
	empty, err := repository.ListCheckResults(ctx, "pr_missing")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty checks = %#v, %v", empty, err)
	}
}

func TestListApprovalsNewestFirst(t *testing.T) {
	ctx := context.Background()
	repository, proposal := proposalRepository(t)
	older := time.Date(2026, 8, 31, 7, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	for _, value := range []ledger.Approval{
		{ID: "apr_older", ProposalID: proposal.ID, ProposalHash: proposal.Hash, Actor: "older", CreatedAt: older},
		{ID: "apr_newer", ProposalID: proposal.ID, ProposalHash: "sha256:stale", Actor: "newer", CreatedAt: newer},
	} {
		if err := repository.SaveApproval(ctx, value); err != nil {
			t.Fatal(err)
		}
	}

	items, err := repository.ListApprovals(ctx, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "apr_newer" || items[1].ID != "apr_older" {
		t.Fatalf("approvals = %#v", items)
	}
	empty, err := repository.ListApprovals(ctx, "pr_missing")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty approvals = %#v, %v", empty, err)
	}
}
