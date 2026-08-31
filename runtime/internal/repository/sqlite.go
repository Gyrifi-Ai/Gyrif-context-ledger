package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/migrations"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	db      *sql.DB
	objects *ObjectStore
}

func OpenSQLite(ctx context.Context, path, objectPath string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure sqlite: %w", err)
		}
	}
	objects, err := NewObjectStore(objectPath)
	if err != nil {
		db.Close()
		return nil, err
	}
	repository := &SQLite{db: db, objects: objects}
	if err := repository.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return repository, nil
}

func (repository *SQLite) migrate(ctx context.Context) error {
	if _, err := repository.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("prepare migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var exists int
		if err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, entry.Name()).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists != 0 {
			continue
		}
		contents, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := repository.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, string(contents)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, entry.Name(), formatTime(time.Now()))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (repository *SQLite) Close() error { return repository.db.Close() }
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func (repository *SQLite) CreateLedger(ctx context.Context, value ledger.Ledger) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO ledgers(id,name,description,created_at) VALUES(?,?,?,?)`, value.ID, value.Name, value.Description, formatTime(value.CreatedAt)); err != nil {
		return fmt.Errorf("insert ledger: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO ledger_heads(ledger_id,release_id) VALUES(?, '')`, value.ID); err != nil {
		return fmt.Errorf("initialize ledger head: %w", err)
	}
	return tx.Commit()
}

func (repository *SQLite) ListLedgers(ctx context.Context) ([]ledger.Ledger, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id,name,description,created_at FROM ledgers ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list ledgers: %w", err)
	}
	defer rows.Close()
	var items []ledger.Ledger
	for rows.Next() {
		var item ledger.Ledger
		var created string
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanChange(scanner interface{ Scan(...any) error }) (ledger.Change, error) {
	var item ledger.Change
	var created string
	var desired []byte
	err := scanner.Scan(&item.ID, &item.LedgerID, &item.Sequence, &item.Unit, &item.Action, &desired, &item.BaseFingerprint, &item.DesiredFingerprint, &item.IdempotencyKey, &item.RequestFingerprint, &item.Status, &created)
	item.Desired = desired
	item.CreatedAt = parseTime(created)
	return item, err
}

const changeColumns = `id,ledger_id,sequence,unit_key,action,desired,base_fingerprint,desired_fingerprint,idempotency_key,request_fingerprint,status,created_at`

func (repository *SQLite) FindChangeByIdempotencyKey(ctx context.Context, ledgerID, key string) (ledger.Change, error) {
	item, err := scanChange(repository.db.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM changes WHERE ledger_id=? AND idempotency_key=?`, ledgerID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return ledger.Change{}, ErrNotFound
	}
	return item, err
}

func (repository *SQLite) InsertChange(ctx context.Context, value *ledger.Change) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM changes WHERE ledger_id=?`, value.LedgerID).Scan(&value.Sequence); err != nil {
		return fmt.Errorf("allocate change sequence: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO changes(id,ledger_id,sequence,unit_key,action,desired,base_fingerprint,desired_fingerprint,idempotency_key,request_fingerprint,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, value.ID, value.LedgerID, value.Sequence, value.Unit, value.Action, []byte(value.Desired), value.BaseFingerprint, value.DesiredFingerprint, value.IdempotencyKey, value.RequestFingerprint, value.Status, formatTime(value.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert change: %w", err)
	}
	return tx.Commit()
}

func (repository *SQLite) ListChanges(ctx context.Context, ledgerID string) ([]ledger.Change, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT `+changeColumns+` FROM changes WHERE ledger_id=? ORDER BY sequence DESC`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ledger.Change
	for rows.Next() {
		item, err := scanChange(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repository *SQLite) LoadChanges(ctx context.Context, ledgerID string, ids []string) ([]ledger.Change, error) {
	items := make([]ledger.Change, 0, len(ids))
	for _, id := range ids {
		item, err := scanChange(repository.db.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM changes WHERE ledger_id=? AND id=?`, ledgerID, id))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (repository *SQLite) InsertProposal(ctx context.Context, value ledger.Proposal) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO proposals(id,ledger_id,title,base_release_id,proposal_hash,status,created_at) VALUES(?,?,?,?,?,?,?)`, value.ID, value.LedgerID, value.Title, value.BaseReleaseID, value.Hash, value.Status, formatTime(value.CreatedAt)); err != nil {
		return fmt.Errorf("insert proposal: %w", err)
	}
	for index, id := range value.ChangeIDs {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM changes WHERE id=? AND ledger_id=?`, id, value.LedgerID).Scan(&status); err != nil {
			return fmt.Errorf("load proposal change: %w", err)
		}
		if status != string(ledger.ChangeReady) {
			return fmt.Errorf("%w: change %s is %s", ledger.ErrInvalid, id, status)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO proposal_changes(proposal_id,change_id,ordinal) VALUES(?,?,?)`, value.ID, id, index); err != nil {
			return fmt.Errorf("%w: %v", ErrChangeClaimed, err)
		}
	}
	return tx.Commit()
}

func scanProposal(scanner interface{ Scan(...any) error }) (ledger.Proposal, error) {
	var item ledger.Proposal
	var created string
	err := scanner.Scan(&item.ID, &item.LedgerID, &item.Title, &item.BaseReleaseID, &item.Hash, &item.Status, &created)
	item.CreatedAt = parseTime(created)
	return item, err
}
func (repository *SQLite) proposalChanges(ctx context.Context, id string) ([]string, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT change_id FROM proposal_changes WHERE proposal_id=? ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		ids = append(ids, value)
	}
	return ids, rows.Err()
}
func (repository *SQLite) LoadProposal(ctx context.Context, ledgerID, id string) (ledger.Proposal, error) {
	item, err := scanProposal(repository.db.QueryRowContext(ctx, `SELECT id,ledger_id,title,base_release_id,proposal_hash,status,created_at FROM proposals WHERE ledger_id=? AND id=?`, ledgerID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.ChangeIDs, err = repository.proposalChanges(ctx, id)
	return item, err
}
func (repository *SQLite) ListProposals(ctx context.Context, ledgerID string) ([]ledger.Proposal, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id,ledger_id,title,base_release_id,proposal_hash,status,created_at FROM proposals WHERE ledger_id=? ORDER BY created_at DESC`, ledgerID)
	if err != nil {
		return nil, err
	}
	var items []ledger.Proposal
	for rows.Next() {
		item, err := scanProposal(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].ChangeIDs, err = repository.proposalChanges(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (repository *SQLite) SaveCheckResult(ctx context.Context, value ledger.CheckResult) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO checks(id,proposal_id,proposal_hash,kind,passed,summary,evidence,created_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.ProposalID, value.ProposalHash, value.Kind, value.Passed, value.Summary, value.Evidence, formatTime(value.CreatedAt)); err != nil {
		return err
	}
	if value.Passed {
		_, err = tx.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=? AND proposal_hash=?`, ledger.ProposalReviewed, value.ProposalID, value.ProposalHash)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=? AND proposal_hash=?`, ledger.ProposalBlocked, value.ProposalID, value.ProposalHash)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (repository *SQLite) ListCheckResults(ctx context.Context, proposalID string) ([]ledger.CheckResult, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id,proposal_id,proposal_hash,kind,passed,summary,evidence,created_at FROM checks WHERE proposal_id=? ORDER BY created_at DESC, id DESC`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ledger.CheckResult, 0)
	for rows.Next() {
		var item ledger.CheckResult
		var created string
		if err := rows.Scan(&item.ID, &item.ProposalID, &item.ProposalHash, &item.Kind, &item.Passed, &item.Summary, &item.Evidence, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLite) HasPassingCheck(ctx context.Context, id, hash string) (bool, error) {
	var count int
	err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM checks WHERE proposal_id=? AND proposal_hash=? AND passed=1`, id, hash).Scan(&count)
	return count > 0, err
}
func (repository *SQLite) SaveApproval(ctx context.Context, value ledger.Approval) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO approvals(id,proposal_id,proposal_hash,actor,created_at) VALUES(?,?,?,?,?) ON CONFLICT(proposal_id,proposal_hash,actor) DO NOTHING`, value.ID, value.ProposalID, value.ProposalHash, value.Actor, formatTime(value.CreatedAt)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=? AND proposal_hash=?`, ledger.ProposalApproved, value.ProposalID, value.ProposalHash); err != nil {
		return err
	}
	return tx.Commit()
}
func (repository *SQLite) ListApprovals(ctx context.Context, proposalID string) ([]ledger.Approval, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id,proposal_id,proposal_hash,actor,created_at FROM approvals WHERE proposal_id=? ORDER BY created_at DESC, id DESC`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ledger.Approval, 0)
	for rows.Next() {
		var item ledger.Approval
		var created string
		if err := rows.Scan(&item.ID, &item.ProposalID, &item.ProposalHash, &item.Actor, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLite) HasApproval(ctx context.Context, id, hash string) (bool, error) {
	var count int
	err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM approvals WHERE proposal_id=? AND proposal_hash=?`, id, hash).Scan(&count)
	return count > 0, err
}
func (repository *SQLite) CurrentHead(ctx context.Context, ledgerID string) (ledger.Head, error) {
	var value ledger.Head
	value.LedgerID = ledgerID
	err := repository.db.QueryRowContext(ctx, `SELECT release_id FROM ledger_heads WHERE ledger_id=?`, ledgerID).Scan(&value.ReleaseID)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}
func (repository *SQLite) SaveReleaseIntent(ctx context.Context, value ledger.ReleaseIntent) error {
	_, err := repository.db.ExecContext(ctx, `INSERT INTO release_intents(id,ledger_id,proposal_id,proposal_hash,parent_id,status,plan,created_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.LedgerID, value.ProposalID, value.ProposalHash, value.ParentID, value.Status, value.Plan, formatTime(value.CreatedAt))
	return err
}
func (repository *SQLite) UpdateReleaseIntent(ctx context.Context, id string, status ledger.ReleaseIntentStatus) error {
	_, err := repository.db.ExecContext(ctx, `UPDATE release_intents SET status=? WHERE id=?`, status, id)
	return err
}
func (repository *SQLite) ResolveReleaseIntent(ctx context.Context, id, note string, resolvedAt time.Time) error {
	result, err := repository.db.ExecContext(ctx, `UPDATE release_intents SET status=?,resolution=?,resolution_note=?,resolved_at=? WHERE id=? AND status=?`, ledger.IntentAbandoned, string(ledger.IntentAbandoned), note, formatTime(resolvedAt), id, ledger.IntentRecoveryRequired)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return ledger.ErrConflict
	}
	return nil
}
func scanIntent(scanner interface{ Scan(...any) error }) (ledger.ReleaseIntent, error) {
	var value ledger.ReleaseIntent
	var created string
	var resolution, resolutionNote, resolvedAt sql.NullString
	err := scanner.Scan(&value.ID, &value.LedgerID, &value.ProposalID, &value.ProposalHash, &value.ParentID, &value.Status, &value.Plan, &created, &resolution, &resolutionNote, &resolvedAt)
	value.CreatedAt = parseTime(created)
	value.Resolution = resolution.String
	value.ResolutionNote = resolutionNote.String
	if resolvedAt.Valid {
		parsed := parseTime(resolvedAt.String)
		value.ResolvedAt = &parsed
	}
	return value, err
}

const releaseIntentColumns = `id,ledger_id,proposal_id,proposal_hash,parent_id,status,plan,created_at,resolution,resolution_note,resolved_at`

func (repository *SQLite) LoadReleaseIntent(ctx context.Context, id string) (ledger.ReleaseIntent, error) {
	value, err := scanIntent(repository.db.QueryRowContext(ctx, `SELECT `+releaseIntentColumns+` FROM release_intents WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}
func (repository *SQLite) ListReleaseIntentsForLedger(ctx context.Context, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ledger.ReleaseIntent, error) {
	query := `SELECT ` + releaseIntentColumns + ` FROM release_intents WHERE ledger_id=?`
	arguments := []any{ledgerID}
	if status != nil {
		query += ` AND status=?`
		arguments = append(arguments, *status)
	}
	query += ` ORDER BY created_at DESC,id DESC`
	rows, err := repository.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ledger.ReleaseIntent, 0)
	for rows.Next() {
		item, err := scanIntent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLite) ListUnfinishedReleaseIntents(ctx context.Context) ([]ledger.ReleaseIntent, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT `+releaseIntentColumns+` FROM release_intents WHERE status NOT IN (?,?)`, ledger.IntentFinalized, ledger.IntentAbandoned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ledger.ReleaseIntent
	for rows.Next() {
		item, err := scanIntent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLite) LoadReleaseIntentForProposal(ctx context.Context, proposalID string) (ledger.ReleaseIntent, error) {
	value, err := scanIntent(repository.db.QueryRowContext(ctx, `SELECT `+releaseIntentColumns+` FROM release_intents WHERE proposal_id=? AND status=? ORDER BY created_at DESC LIMIT 1`, proposalID, ledger.IntentFinalized))
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}
func (repository *SQLite) FinalizeRelease(ctx context.Context, intent ledger.ReleaseIntent, value ledger.Release) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var head string
	if err = tx.QueryRowContext(ctx, `SELECT release_id FROM ledger_heads WHERE ledger_id=?`, intent.LedgerID).Scan(&head); err != nil {
		return err
	}
	if head != intent.ParentID {
		return ledger.ErrConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO releases(id,ledger_id,proposal_id,parent_id,release_hash,created_at) VALUES(?,?,?,?,?,?)`, value.ID, value.LedgerID, value.ProposalID, value.ParentID, value.Hash, formatTime(value.CreatedAt)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ledger_heads SET release_id=? WHERE ledger_id=?`, value.ID, value.LedgerID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=?`, ledger.ProposalReleased, intent.ProposalID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE changes SET status=? WHERE id IN (SELECT change_id FROM proposal_changes WHERE proposal_id=?)`, ledger.ChangeReleased, intent.ProposalID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE release_intents SET status=? WHERE id=?`, ledger.IntentFinalized, intent.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (repository *SQLite) ListReleases(ctx context.Context, ledgerID string) ([]ledger.Release, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id,ledger_id,proposal_id,parent_id,release_hash,created_at FROM releases WHERE ledger_id=? ORDER BY created_at DESC`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ledger.Release
	for rows.Next() {
		var item ledger.Release
		var created string
		if err := rows.Scan(&item.ID, &item.LedgerID, &item.ProposalID, &item.ParentID, &item.Hash, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLite) WriteObject(ctx context.Context, kind string, value []byte) (string, error) {
	return repository.objects.Write(ctx, kind, value)
}
func (repository *SQLite) ReadObject(ctx context.Context, hash string) ([]byte, error) {
	return repository.objects.Read(ctx, hash)
}
