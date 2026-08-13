# GRF-222 — Retention budgets, quotas, and backup command

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Operations |
| Priority | High |
| Size | L |
| Depends on | — |
| Blocks | — |

## Summary

Bound the runtime's disk growth and give operators a supported way to back up and restore `/data`. Today the object store grows without limit and there is no backup procedure.

## Context

Every Change writes its desired value into the content-addressed object store at `<data>/objects/<aa>/<hash>`. Every Release retains before-images for rollback. Nothing is ever removed. A busy ingestion pipeline writing large embedding vectors will fill the volume, and when it does:

- SQLite writes begin failing,
- an in-flight release can fail after apply but before verify, producing `RECOVERY_REQUIRED`,
- rollback material may be unwritable, which silently undermines the product's core guarantee.

There is also no backup story. `/data` holds `gyrifi.db`, `gyrifi.db-wal`, `gyrifi.db-shm`, and `objects/`. Copying those files while the runtime is writing produces a corrupt or inconsistent snapshot.

## Scope

### In scope

- Quota enforcement that rejects ingestion **before** the filesystem fills.
- A retention policy for objects that are no longer reachable.
- `gyrifi backup` producing a consistent, restorable snapshot.
- Disk usage reporting in `/api/v1/system/status`.

### Out of scope

- Remote backup destinations (S3 etc.). Produce a local artefact; let operators ship it.
- Point-in-time recovery.
- Compressing or deduplicating vector payloads.

## Reachability — get this right or lose rollback

An object is **reachable** if any of the following reference its hash:

1. a `changes` row's desired value,
2. a `release_intents` plan's expected or before-image values, for any intent not `FINALIZED` **or** for any `FINALIZED` intent belonging to a release at or after the oldest release still within the rollback window,
3. a `checks` evidence blob.

Deleting a reachable object destroys the ability to roll back. **Retention must be conservative: when reachability is uncertain, keep the object.**

Because rollback is forward-only and reconstructs state from before-images, the rollback window is defined by *releases*, not by time alone. The default policy is "retain rollback material for the most recent N releases per ledger", not "delete objects older than X days".

## Acceptance criteria

**Quotas**

- [ ] New config: `GYRIFI_DATA_QUOTA_BYTES` (default `0` = unlimited), `GYRIFI_ROLLBACK_WINDOW_RELEASES` (default `50`), `GYRIFI_MAX_OBJECT_BYTES` (default `4 MiB`, matching the existing request body cap).
- [ ] A single desired value exceeding `GYRIFI_MAX_OBJECT_BYTES` ⇒ `400 INVALID_ARGUMENT` before anything is written.
- [ ] When measured usage exceeds the quota, `POST .../changes` returns `507 INSUFFICIENT_STORAGE` (or `503 UNAVAILABLE` with a storage code if 507 is undesirable — decide and document) with a message naming the quota.
- [ ] **Release and rollback are never blocked by the quota.** Governance operations on already-ingested data must complete; only new ingestion is rejected. A test asserts this explicitly.
- [ ] Usage is measured incrementally (a running total maintained on write/delete), not by walking the object tree on every request.
- [ ] The running total is reconciled by a full walk at startup and recorded in the log.

**Retention**

- [ ] A retention pass runs on startup and on a configurable interval (`GYRIFI_RETENTION_INTERVAL`, default `1h`, `0` disables).
- [ ] The pass computes the reachable set from the database, then removes only unreachable objects.
- [ ] Objects newer than a grace period (default 15 minutes) are never removed, so a concurrently-written object cannot be collected before its referencing row commits.
- [ ] Removal never runs concurrently with itself.
- [ ] Objects referenced by an intent in `RECOVERY_REQUIRED` or `ABANDONED` are always retained.
- [ ] The pass logs objects examined, retained, removed, and bytes reclaimed.
- [ ] A dry-run mode (`gyrifi retention --dry-run`) reports what would be removed without removing it.

**Backup**

- [ ] `gyrifi backup --out <path>` produces a directory or tar containing a consistent SQLite copy plus every object reachable from it.
- [ ] The SQLite copy uses the backup API / `VACUUM INTO`, **not** a file copy, so WAL state is consistent.
- [ ] The object set is computed from the snapshot, not from the live database.
- [ ] The artefact includes a `manifest.json` with the schema version, the runtime version, the ledger ids, the object count, and a checksum per object.
- [ ] `gyrifi backup --verify <path>` re-checks every object checksum and that every hash referenced by the snapshot is present.
- [ ] Backup runs against a live runtime without blocking writes for longer than the SQLite backup step requires.
- [ ] Restore is documented as: stop the runtime, replace `/data`, start. A test performs exactly this and then completes a release.

**Reporting**

- [ ] `GET /api/v1/system/status` reports `{ "storage": { "objectBytes", "databaseBytes", "quotaBytes", "objectCount" } }`.
- [ ] Studio shows storage usage in the shell footer or `#settings`, with an amber state above 80% and a danger state above 95% of the quota.
- [ ] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- Reachability computation belongs in `runtime/internal/repository/retention.go` since it is fundamentally a query over persisted references.
- Compute the reachable set as a `map[string]struct{}` of hashes. For the expected data volumes this fits comfortably in memory; if it ever does not, that is a separate ticket, not a reason to build a mark-and-sweep now.
- Walk the object tree once per pass, streaming; do not load object contents.
- Use `VACUUM INTO 'path'` for the snapshot — it is supported by `modernc.org/sqlite` and yields a consistent single-file copy.
- Delete objects with `os.Remove` and prune empty shard directories opportunistically; ignore `ErrNotExist`.
- The grace period is the defence against the write-then-commit race. Do not replace it with a lock.
- Never delete an object because it is "old". Age is only ever a safety floor, never a reason.

## Test plan

- `runtime/internal/repository/retention_test.go`:
  - an object referenced only by a change is retained,
  - an object referenced only by a finalized intent outside the rollback window is removed,
  - the same object inside the window is retained,
  - an object referenced by a `RECOVERY_REQUIRED` intent is always retained,
  - an object written 1 minute ago with no references is retained (grace period),
  - the same object aged past the grace period is removed.
- `runtime/tests/quota_test.go`:
  - ingestion beyond the quota is rejected,
  - release and rollback still succeed while over quota,
  - an oversized value is rejected before any write.
- `runtime/tests/backup_test.go`:
  - backup a live runtime, verify the artefact,
  - restore into a fresh data directory and complete a release and a rollback from the restored state,
  - a tampered object fails `--verify`.
- Startup reconciliation matches a manual walk.

## Docs to update

- `docs/ai/tech-spec.md` §2 (config), §3 (status payload, new error code), §7 (object store lifecycle and reachability rules).
- `docs/ai/product.md` §5 — retention as an invariant; §7 — remove the retention gap row.
- `README.md` — backup/restore procedure and quota configuration.
- `docs/ai/phases/phase-3.md` — completion entry recording the exact reachability definition implemented.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
