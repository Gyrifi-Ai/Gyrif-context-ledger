package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/migrations"
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

func TestListChangesScansNullDesiredForDelete(t *testing.T) {
	ctx := context.Background()
	repository, proposal := proposalRepository(t)
	change := ledger.Change{ID: "chg_delete", LedgerID: proposal.LedgerID, Unit: "obsolete", Action: ledger.ChangeDelete, DesiredFingerprint: ledger.Fingerprint(nil), IdempotencyKey: "delete", RequestFingerprint: "sha256:delete", Status: ledger.ChangeReady, CreatedAt: time.Now().UTC()}
	if err := repository.InsertChange(ctx, &change); err != nil {
		t.Fatal(err)
	}

	items, err := repository.ListChanges(ctx, proposal.LedgerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != change.ID || items[0].Desired != nil {
		t.Fatalf("changes = %#v", items)
	}
	encoded, err := json.Marshal(items[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"desired"`) {
		t.Fatalf("DELETE response exposed desired: %s", encoded)
	}
}

func TestCancelProposalRetainsSnapshotAndReleasesClaim(t *testing.T) {
	ctx := context.Background()
	repository, proposal := proposalRepository(t)
	if err := repository.SaveCheckResult(ctx, ledger.CheckResult{ID: "chk_cancelled", ProposalID: proposal.ID, ProposalHash: proposal.Hash, Kind: "deterministic", Passed: true, Summary: "retained", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveApproval(ctx, ledger.Approval{ID: "apr_cancelled", ProposalID: proposal.ID, ProposalHash: proposal.Hash, Actor: "reviewer", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=?`, ledger.ProposalDraft, proposal.ID); err != nil {
		t.Fatal(err)
	}
	if err := repository.CancelProposal(ctx, proposal.LedgerID, proposal.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.LoadProposal(ctx, proposal.LedgerID, proposal.ID)
	if err != nil || loaded.Status != ledger.ProposalCancelled || len(loaded.ChangeIDs) != 1 || loaded.ChangeIDs[0] != proposal.ChangeIDs[0] {
		t.Fatalf("cancelled Proposal = %#v, %v", loaded, err)
	}
	var claims int
	if err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM proposal_changes WHERE proposal_id=?`, proposal.ID).Scan(&claims); err != nil || claims != 0 {
		t.Fatalf("claims = %d, %v", claims, err)
	}
	replacement := proposal
	replacement.ID = "pr_replacement"
	replacement.Title = "Replacement"
	replacement.Status = ledger.ProposalDraft
	if err := repository.InsertProposal(ctx, replacement); err != nil {
		t.Fatalf("re-propose released Change: %v", err)
	}
	checks, err := repository.ListCheckResults(ctx, proposal.ID)
	if err != nil || len(checks) != 1 || checks[0].ID != "chk_cancelled" {
		t.Fatalf("retained checks = %#v, %v", checks, err)
	}
	approvals, err := repository.ListApprovals(ctx, proposal.ID)
	if err != nil || len(approvals) != 1 || approvals[0].ID != "apr_cancelled" {
		t.Fatalf("retained approvals = %#v, %v", approvals, err)
	}
	if err := repository.CancelProposal(ctx, proposal.LedgerID, proposal.ID); !errors.Is(err, ErrProposalAlreadyCancelled) {
		t.Fatalf("second cancellation = %v", err)
	}
}

func TestProposalCancellationMigrationBackfillsOrderedSnapshot(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	path := filepath.Join(directory, "state.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := migrations.Files.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(initial)); err != nil {
		t.Fatal(err)
	}
	created := formatTime(time.Now().UTC())
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, []any{"001_initial.sql", created}},
		{`INSERT INTO ledgers(id,name,created_at) VALUES(?,?,?)`, []any{"ldg_upgrade", "Upgrade", created}},
		{`INSERT INTO ledger_heads(ledger_id,release_id) VALUES(?,?)`, []any{"ldg_upgrade", ""}},
		{`INSERT INTO changes(id,ledger_id,sequence,unit_key,action,desired,desired_fingerprint,idempotency_key,request_fingerprint,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, []any{"chg_first", "ldg_upgrade", 1, "first", "PUT", []byte(`{"value":1}`), "sha256:first", "first", "sha256:first-request", "READY", created}},
		{`INSERT INTO changes(id,ledger_id,sequence,unit_key,action,desired,desired_fingerprint,idempotency_key,request_fingerprint,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, []any{"chg_second", "ldg_upgrade", 2, "second", "PUT", []byte(`{"value":2}`), "sha256:second", "second", "sha256:second-request", "READY", created}},
		{`INSERT INTO proposals(id,ledger_id,title,proposal_hash,status,created_at) VALUES(?,?,?,?,?,?)`, []any{"pr_upgrade", "ldg_upgrade", "Upgrade Proposal", "sha256:proposal", "DRAFT", created}},
		{`INSERT INTO proposal_changes(proposal_id,change_id,ordinal) VALUES(?,?,?)`, []any{"pr_upgrade", "chg_second", 0}},
		{`INSERT INTO proposal_changes(proposal_id,change_id,ordinal) VALUES(?,?,?)`, []any{"pr_upgrade", "chg_first", 1}},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := OpenSQLite(ctx, path, filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	proposal, err := upgraded.LoadProposal(ctx, "ldg_upgrade", "pr_upgrade")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(proposal.ChangeIDs, ","), "chg_second,chg_first"; got != want {
		t.Fatalf("backfilled Change order = %q, want %q", got, want)
	}
	items, err := upgraded.ListProposals(ctx, "ldg_upgrade")
	if err != nil || len(items) != 1 || strings.Join(items[0].ChangeIDs, ",") != "chg_second,chg_first" {
		t.Fatalf("upgraded Proposal list = %#v, %v", items, err)
	}
}

func TestReleaseIntentResolutionAndListing(t *testing.T) {
	ctx := context.Background()
	repository, proposal := proposalRepository(t)
	older := ledger.ReleaseIntent{ID: "intent_older", LedgerID: proposal.LedgerID, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Status: ledger.IntentRecoveryRequired, Plan: []byte(`{"operations":[]}`), CreatedAt: time.Now().UTC().Add(-time.Minute)}
	newer := ledger.ReleaseIntent{ID: "intent_newer", LedgerID: proposal.LedgerID, ProposalID: proposal.ID, ProposalHash: proposal.Hash, Status: ledger.IntentVerifying, Plan: []byte(`{"operations":[]}`), CreatedAt: time.Now().UTC()}
	for _, intent := range []ledger.ReleaseIntent{older, newer} {
		if err := repository.SaveReleaseIntent(ctx, intent); err != nil {
			t.Fatal(err)
		}
	}
	items, err := repository.ListReleaseIntentsForLedger(ctx, proposal.LedgerID, nil)
	if err != nil || len(items) != 2 || items[0].ID != newer.ID {
		t.Fatalf("release intents = %#v, %v", items, err)
	}
	status := ledger.IntentRecoveryRequired
	items, err = repository.ListReleaseIntentsForLedger(ctx, proposal.LedgerID, &status)
	if err != nil || len(items) != 1 || items[0].ID != older.ID {
		t.Fatalf("filtered release intents = %#v, %v", items, err)
	}
	resolvedAt := time.Now().UTC()
	if err := repository.ResolveReleaseIntent(ctx, older.ID, "Operator inspected target", resolvedAt); err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.LoadReleaseIntent(ctx, older.ID)
	if err != nil || loaded.Status != ledger.IntentAbandoned || loaded.Resolution != string(ledger.IntentAbandoned) || loaded.ResolutionNote != "Operator inspected target" || loaded.ResolvedAt == nil {
		t.Fatalf("resolved intent = %#v, %v", loaded, err)
	}
	if err := repository.ResolveReleaseIntent(ctx, older.ID, "again", time.Now()); err == nil {
		t.Fatal("second resolution should conflict")
	}
	unfinished, err := repository.ListUnfinishedReleaseIntents(ctx)
	if err != nil || len(unfinished) != 1 || unfinished[0].ID != newer.ID {
		t.Fatalf("unfinished intents = %#v, %v", unfinished, err)
	}
}
