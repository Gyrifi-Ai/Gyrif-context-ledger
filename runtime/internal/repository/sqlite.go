package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
	db                 *sql.DB
	objects            *ObjectStore
	expectedMigrations []string
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
	migrationCount, err := repository.migrate(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}
	repository.expectedMigrations = migrationCount
	return repository, nil
}

func (repository *SQLite) migrate(ctx context.Context) ([]string, error) {
	if _, err := repository.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return nil, fmt.Errorf("prepare migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versions = append(versions, entry.Name())
		var exists int
		if err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, entry.Name()).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists != 0 {
			continue
		}
		contents, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := repository.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, string(contents)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, entry.Name(), formatTime(time.Now()))
		}
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return versions, nil
}

func (repository *SQLite) Close() error { return repository.db.Close() }
func (repository *SQLite) Readiness(ctx context.Context) (bool, error) {
	if len(repository.expectedMigrations) == 0 {
		return false, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(repository.expectedMigrations)), ",")
	args := make([]any, len(repository.expectedMigrations))
	for index, version := range repository.expectedMigrations {
		args[index] = version
	}
	var applied int
	if err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version IN (`+placeholders+`)`, args...).Scan(&applied); err != nil {
		return false, err
	}
	return applied == len(repository.expectedMigrations), nil
}
func (repository *SQLite) DatabaseStats(ctx context.Context) (OperationalStats, error) {
	var stats OperationalStats
	if err := repository.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM release_intents WHERE status=?),
		(SELECT COUNT(*) FROM changes WHERE status IN (?,?))`,
		ledger.IntentRecoveryRequired, ledger.ChangeAccepted, ledger.ChangeReady,
	).Scan(&stats.UnresolvedIntents, &stats.PendingChanges); err != nil {
		return OperationalStats{}, err
	}
	return stats, nil
}
func (repository *SQLite) ObjectStoreBytes(ctx context.Context) (int64, error) {
	return repository.objects.Size(ctx)
}
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

func (repository *SQLite) ListLedgers(ctx context.Context, options ListOptions) (Page[ledger.Ledger], error) {
	query := `SELECT id,name,description,created_at FROM ledgers`
	args := make([]any, 0, 3)
	if options.Cursor != nil {
		query += ` WHERE (created_at, id) < (?, ?)`
		args = append(args, formatTime(options.Cursor.CreatedAt), options.Cursor.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, options.Limit+1)
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page[ledger.Ledger]{}, fmt.Errorf("list ledgers: %w", err)
	}
	defer rows.Close()
	items := make([]ledger.Ledger, 0, options.Limit+1)
	for rows.Next() {
		var item ledger.Ledger
		var created string
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &created); err != nil {
			return Page[ledger.Ledger]{}, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[ledger.Ledger]{}, err
	}
	page := Page[ledger.Ledger]{Items: items, HasMore: len(items) > options.Limit}
	if page.HasMore {
		page.Items = page.Items[:options.Limit]
	}
	return page, nil
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

func (repository *SQLite) ListChanges(ctx context.Context, ledgerID string, options ListOptions) (Page[ledger.Change], error) {
	query := `SELECT ` + changeColumns + ` FROM changes WHERE ledger_id=?`
	args := []any{ledgerID}
	if options.Status != nil {
		query += ` AND status=?`
		args = append(args, *options.Status)
	}
	if options.Action != nil {
		query += ` AND action=?`
		args = append(args, *options.Action)
	}
	if options.Cursor != nil {
		query += ` AND (created_at, id) < (?, ?)`
		args = append(args, formatTime(options.Cursor.CreatedAt), options.Cursor.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, options.Limit+1)
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page[ledger.Change]{}, err
	}
	defer rows.Close()
	items := make([]ledger.Change, 0, options.Limit+1)
	for rows.Next() {
		item, err := scanChange(rows)
		if err != nil {
			return Page[ledger.Change]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[ledger.Change]{}, err
	}
	page := Page[ledger.Change]{Items: items, HasMore: len(items) > options.Limit}
	if page.HasMore {
		page.Items = page.Items[:options.Limit]
	}
	return page, nil
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
	changeIDs, err := json.Marshal(value.ChangeIDs)
	if err != nil {
		return fmt.Errorf("encode proposal Changes: %w", err)
	}
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO proposals(id,ledger_id,title,base_release_id,proposal_hash,status,change_ids,created_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.LedgerID, value.Title, value.BaseReleaseID, value.Hash, value.Status, string(changeIDs), formatTime(value.CreatedAt)); err != nil {
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
	var changeIDs, created string
	err := scanner.Scan(&item.ID, &item.LedgerID, &item.Title, &item.BaseReleaseID, &item.Hash, &item.Status, &changeIDs, &created)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal([]byte(changeIDs), &item.ChangeIDs); err != nil {
		return item, fmt.Errorf("decode proposal Changes: %w", err)
	}
	item.CreatedAt = parseTime(created)
	return item, nil
}
func (repository *SQLite) LoadProposal(ctx context.Context, ledgerID, id string) (ledger.Proposal, error) {
	item, err := scanProposal(repository.db.QueryRowContext(ctx, `SELECT id,ledger_id,title,base_release_id,proposal_hash,status,change_ids,created_at FROM proposals WHERE ledger_id=? AND id=?`, ledgerID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}
func (repository *SQLite) ListProposals(ctx context.Context, ledgerID string, options ListOptions) (Page[ledger.Proposal], error) {
	query := `SELECT id,ledger_id,title,base_release_id,proposal_hash,status,change_ids,created_at FROM proposals WHERE ledger_id=?`
	args := []any{ledgerID}
	if options.Status != nil {
		query += ` AND status=?`
		args = append(args, *options.Status)
	}
	if options.Cursor != nil {
		query += ` AND (created_at, id) < (?, ?)`
		args = append(args, formatTime(options.Cursor.CreatedAt), options.Cursor.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, options.Limit+1)
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page[ledger.Proposal]{}, err
	}
	defer rows.Close()
	items := make([]ledger.Proposal, 0, options.Limit+1)
	for rows.Next() {
		item, err := scanProposal(rows)
		if err != nil {
			return Page[ledger.Proposal]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[ledger.Proposal]{}, err
	}
	page := Page[ledger.Proposal]{Items: items, HasMore: len(items) > options.Limit}
	if page.HasMore {
		page.Items = page.Items[:options.Limit]
	}
	return page, nil
}

func (repository *SQLite) HasReleaseIntent(ctx context.Context, proposalID string) (bool, error) {
	var count int
	err := repository.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM release_intents WHERE proposal_id=?`, proposalID).Scan(&count)
	return count > 0, err
}

func (repository *SQLite) CancelProposal(ctx context.Context, ledgerID, proposalID string) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status ledger.ProposalStatus
	if err := tx.QueryRowContext(ctx, `SELECT status FROM proposals WHERE ledger_id=? AND id=?`, ledgerID, proposalID).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if status == ledger.ProposalCancelled {
		return ErrProposalAlreadyCancelled
	}
	if err := ledger.CanCancelProposal(ledger.Proposal{Status: status}); err != nil {
		if errors.Is(err, ledger.ErrConflict) {
			if status == ledger.ProposalReleased {
				return ErrProposalReleased
			}
			return ErrProposalNotDraft
		}
		return err
	}
	var intents int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM release_intents WHERE proposal_id=?`, proposalID).Scan(&intents); err != nil {
		return err
	}
	if intents != 0 {
		return ErrProposalReleaseStarted
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM proposal_changes WHERE proposal_id=?`, proposalID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE proposals SET status=? WHERE ledger_id=? AND id=?`, ledger.ProposalCancelled, ledgerID, proposalID); err != nil {
		return err
	}
	return tx.Commit()
}

func (repository *SQLite) SaveCheckResult(ctx context.Context, value ledger.CheckResult) error {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status ledger.ProposalStatus
	if err := tx.QueryRowContext(ctx, `SELECT status FROM proposals WHERE id=?`, value.ProposalID).Scan(&status); err != nil {
		return err
	}
	if status == ledger.ProposalCancelled {
		return ErrProposalAlreadyCancelled
	}
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
	var status ledger.ProposalStatus
	if err := tx.QueryRowContext(ctx, `SELECT status FROM proposals WHERE id=?`, value.ProposalID).Scan(&status); err != nil {
		return err
	}
	if status == ledger.ProposalCancelled {
		return ErrProposalAlreadyCancelled
	}
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
func (repository *SQLite) ListReleases(ctx context.Context, ledgerID string, options ListOptions) (Page[ledger.Release], error) {
	query := `SELECT id,ledger_id,proposal_id,parent_id,release_hash,created_at FROM releases WHERE ledger_id=?`
	args := []any{ledgerID}
	if options.Cursor != nil {
		query += ` AND (created_at, id) < (?, ?)`
		args = append(args, formatTime(options.Cursor.CreatedAt), options.Cursor.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, options.Limit+1)
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page[ledger.Release]{}, err
	}
	defer rows.Close()
	items := make([]ledger.Release, 0, options.Limit+1)
	for rows.Next() {
		var item ledger.Release
		var created string
		if err := rows.Scan(&item.ID, &item.LedgerID, &item.ProposalID, &item.ParentID, &item.Hash, &created); err != nil {
			return Page[ledger.Release]{}, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[ledger.Release]{}, err
	}
	page := Page[ledger.Release]{Items: items, HasMore: len(items) > options.Limit}
	if page.HasMore {
		page.Items = page.Items[:options.Limit]
	}
	return page, nil
}
func (repository *SQLite) WriteObject(ctx context.Context, kind string, value []byte) (string, error) {
	return repository.objects.Write(ctx, kind, value)
}
func (repository *SQLite) ReadObject(ctx context.Context, hash string) ([]byte, error) {
	return repository.objects.Read(ctx, hash)
}
