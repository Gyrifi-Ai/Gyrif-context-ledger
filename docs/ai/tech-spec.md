# Technical specification

Status: extracted from source. Every identifier, path, and default below appears verbatim in the code. If you change the code, change this file in the same commit.

---

## 1. Stack

| Area | Decision | Version |
|---|---|---|
| Runtime | Go modular monolith | Go 1.24 |
| Persistence | SQLite, WAL | `modernc.org/sqlite v1.36.3` (pure Go, `CGO_ENABLED=0`) |
| Large values | Local SHA-256 content-addressed objects | stdlib |
| Frontend | React + TypeScript | React 19.1.1, TS 5.9.2 |
| Frontend build | Vite | 7.1.4 |
| Frontend tests | Vitest | 3.2.4 |
| Package manager | pnpm | 11.15.1 |
| Target adapter | Qdrant REST | — |
| Optional inference | `llama-server` over loopback HTTP | `ghcr.io/ggml-org/llama.cpp:server` |
| Distribution | one shipping Docker image; local Compose stack on loopback | `ubuntu:24.04` base |

Go direct dependency list is exactly one module. Frontend runtime dependency list is exactly `react` and `react-dom`. Adding to either requires an ADR.

---

## 2. Process lifecycle

`cmd/gyrifi/main.go` builds a signal-cancelled context and calls `bootstrap.Run(ctx, os.Args[1:])`.

`bootstrap.Run` order (`internal/bootstrap/bootstrap.go`, `const Version = "0.1.0"`):

1. `config.Load()` — fail fast on invalid env.
2. `os.MkdirAll(DataDirectory, 0o750)` and `os.MkdirAll(dir(SQLitePath), 0o750)`.
3. Build `slog` JSON logger to stdout at `info` or `debug`.
4. `repository.OpenSQLite(ctx, SQLitePath, ObjectsPath)` — opens the DB, applies pragmas, runs migrations, creates the object store.
5. `qdrant.New(QdrantURL, QdrantCollection, QdrantAPIKey)`.
6. If `EvaluationProvider == "llamacpp"`: `os.Stat(ModelPath)`, then `inference.StartLlamaServer(...)`; `defer llama.Stop()`.
7. `engine.New(repo, target, provider)` — `provider` is a nil interface when inference is disabled.
8. `cli.Run(ctx, args, application, os.Stdout)` — **if it handled the args, return here; the HTTP server never starts.**
9. `application.RecoverReleases(ctx)` — logs and continues on error.
10. Construct `httpinterface.New(application, logger, Version)`.
11. `http.Server{ReadHeaderTimeout: 10s, ReadTimeout: 30s, WriteTimeout: 2m, IdleTimeout: 2m}` and `ListenAndServe`.
12. On `ctx.Done()`: `server.Shutdown` with a 10s timeout; deferred closes stop llama-server and the repository.

### Local Docker Compose contract

`compose.yaml` is the supported local launch: `docker compose up --build` starts the `gyrifi` application image, a pinned `qdrant/qdrant:v1.13.4` target, and a short-lived pinned `curlimages/curl:8.12.1` initializer. The initializer waits for Qdrant, then creates collection `gyrifi` with `{ "vectors": { "size": 3, "distance": "Cosine" } }` only if it is absent; it never replaces an existing collection.

`gyrifi` receives `GYRIFI_QDRANT_URL=http://qdrant:6333` and `GYRIFI_QDRANT_COLLECTION=gyrifi`. Its `8080` port is published as `127.0.0.1:8080` only. Named `gyrifi-data` and `qdrant-data` volumes persist both stores across `docker compose down`; `docker compose down --volumes` is the explicit destructive local reset. This is local-only until GRF-220 authenticates the runtime.

### Configuration (`internal/config/config.go`)

| Env var | Default | Validation |
|---|---|---|
| `GYRIFI_HTTP_ADDRESS` | `:8080` | — |
| `GYRIFI_DATA_DIR` | `/data` | — |
| `GYRIFI_SQLITE_PATH` | `{DATA_DIR}/state.db` | — |
| `GYRIFI_OBJECTS_PATH` | `{DATA_DIR}/objects` | — |
| `GYRIFI_QDRANT_URL` | `http://127.0.0.1:6333` | must parse with scheme + host |
| `GYRIFI_QDRANT_COLLECTION` | `gyrifi` | non-empty |
| `GYRIFI_QDRANT_API_KEY` | `""` | read with `os.Getenv` (not trimmed) |
| `GYRIFI_EVALUATION_PROVIDER` | `disabled` | lowercased; must be `disabled` or `llamacpp` |
| `GYRIFI_MODEL_PATH` | `""` | required when provider is `llamacpp` |
| `GYRIFI_LLAMA_SERVER_PATH` | `llama-server` | image sets `/opt/llama/llama-server` |
| `GYRIFI_LLAMA_SERVER_PORT` | `8081` | integer in `1..65535` |
| `GYRIFI_LOG_LEVEL` | `info` | lowercased; `debug` raises the slog level |

Empty-or-whitespace values fall back to the default (helper `environment(name, fallback)`).

### CLI (`internal/interfaces/cli/cli.go`)

Signature: `Run(ctx, args []string, *engine.Engine, io.Writer) (handled bool, err error)`.

| Args | Behaviour |
|---|---|
| *(none)* | `handled = false` → HTTP server starts |
| `doctor` | prints `{"status":"ok","ledgers":N,"inference":"…"}` |
| `version` / `--version` / `-version` | prints `gyrifi dev` — **inconsistent with `bootstrap.Version`, see GRF-223** |
| anything else | error `unknown command "…"` |

---

## 3. HTTP API

Base: `/api/v1`. Router is `http.ServeMux` with Go 1.22 method+pattern syntax and `request.PathValue(...)`.

### Middleware (applies to every route)

- Sets `X-Request-ID: req-{n}` from an `atomic.Uint64` counter.
- Sets `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`.
- Logs method, path, request id, duration.

Request bodies are capped at **4 MiB** (`http.MaxBytesReader`) and decoded with `DisallowUnknownFields()`. Any decode failure ⇒ `400 INVALID_ARGUMENT` with message `Request body is invalid.`

### Error envelope

```json
{ "error": { "code": "CONFLICT", "message": "Current passing evidence and approval are required." } }
```

| `engine.ErrorCode` | HTTP status |
|---|---|
| `INVALID_ARGUMENT` | 400 |
| `NOT_FOUND` | 404 |
| `CONFLICT` | 409 |
| `UNAVAILABLE` | 503 |
| `INTERNAL` | 500 |

`engine.PublicError(err)` returns the wrapped code and message; unwrapped errors degrade to `INTERNAL` / `The operation could not be completed.` Raw SQLite, Qdrant, and llama-server errors are logged, never returned.

### Endpoints

| Method | Path | Success | Request body | Response body |
|---|---|---|---|---|
| GET | `/api/v1/system/status` | 200 | — | `{ "status":"ok", "version":"0.1.0", "inference":"disabled"\|"llamacpp" }` |
| GET | `/api/v1/adapters` | 200 | — | `{ "items":[{ "id":"qdrant", "name":"Qdrant", "capabilities": Capabilities }] }` |
| GET | `/api/v1/ledgers` | 200 | — | `{ "items": Ledger[] }` |
| POST | `/api/v1/ledgers` | 201 | `{ "name", "description" }` | `Ledger` |
| GET | `/api/v1/ledgers/{ledgerID}/changes` | 200 | — | `{ "items": Change[] }` |
| POST | `/api/v1/ledgers/{ledgerID}/changes` | **202** | `{ "unit", "action", "desired", "idempotencyKey" }` | `Change` |
| GET | `/api/v1/ledgers/{ledgerID}/proposals` | 200 | — | `{ "items": Proposal[] }` |
| POST | `/api/v1/ledgers/{ledgerID}/proposals` | 201 | `{ "title", "changeIds":[…] }` | `Proposal` |
| GET | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}` | 200 | — | `ProposalDetail` |
| GET | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}/checks` | 200 | — | `{ "items": CheckResult[] }` (newest first) |
| GET | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals` | 200 | — | `{ "items": Approval[] }` (newest first) |
| POST | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}/evaluation` | 200 | `{ "criteria" }` | `{ "passed", "summary", "previewFidelity", "findings"? }` |
| POST | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals` | **204** | `{ "actor" }` | — |
| POST | `/api/v1/ledgers/{ledgerID}/proposals/{proposalID}/release` | 201 | — | `Release` |
| GET | `/api/v1/ledgers/{ledgerID}/release-intents` | 200 | — | `{ "items": ReleaseIntent[] }` (newest first; optional `?status=`) |
| GET | `/api/v1/ledgers/{ledgerID}/release-intents/{intentID}` | 200 | — | `ReleaseIntent` with expanded Plan and per-operation `hasBeforeImage` |
| POST | `/api/v1/ledgers/{ledgerID}/release-intents/{intentID}/retry` | 200 | — | `{ "resolved": bool, "mismatches": VerificationMismatch[] }` |
| POST | `/api/v1/ledgers/{ledgerID}/release-intents/{intentID}/resolve` | **204** | `{ "resolution":"ABANDONED", "note":string }` | — |
| GET | `/api/v1/ledgers/{ledgerID}/releases` | 200 | — | `{ "items": Release[] }` (newest first) |
| POST | `/api/v1/ledgers/{ledgerID}/releases/{releaseID}/rollback` | 201 | — | `Proposal` |
| GET | `/events/v1` | 200 | — | `text/event-stream` |

Unknown paths under `/api/` or `/events/` return `404 NOT_FOUND`. Everything else falls through to the embedded Studio file server, which serves `index.html` for unmatched paths (SPA fallback).

### `/events/v1`

The stream sends this backwards-compatible frame once after subscribing:

```text
event: ledger
data: {"status":"connected"}
```

then `: keepalive` every 20 seconds until the request context is cancelled. An optional `?ledgerId=ldg_…` query parameter limits domain frames to that Ledger. Each subscriber has a 16-event buffer; a full buffer drops that subscriber's event rather than blocking a governance operation. Events are advisory refetch hints, not a replayable source of truth; `Last-Event-ID` is not supported.

```text
event: proposal.created
data: {"kind":"proposal.created","ledgerId":"ldg_…","subjectId":"pr_…","at":"2026-08-31T07:00:00Z"}
```

| Event kind | Durable transition | `subjectId` |
|---|---|---|
| `change.accepted` | a new Change is inserted; idempotent replays do not emit | Change ID |
| `proposal.created` | a Proposal and its ordered claims are inserted | Proposal ID |
| `proposal.evaluated` | evaluation evidence is saved | Proposal ID |
| `proposal.approved` | approval bound to the current hash is saved | Proposal ID |
| `release.started` | a Release Intent is saved | Intent ID |
| `release.completed` | finalization advances HEAD, including successful startup or operator recovery | Release ID |
| `release.failed` | apply or verification fails and requires recovery | Intent ID |
| `intent.recovery_required` | release, startup recovery, or retry durably marks an Intent for operator recovery | Intent ID |
| `intent.resolved` | retry finalizes or an operator abandons an Intent | Intent ID |

---

## 4. Wire types

Go structs in `internal/ledger` are serialised directly; the TypeScript mirrors live in `studio/src/api/types.ts`.

```go
type Ledger struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"createdAt"`
}

type Head struct {
    LedgerID  string `json:"ledgerId"`
    ReleaseID string `json:"releaseId,omitempty"`
}

type Change struct {
    ID                 string          `json:"id"`
    LedgerID           string          `json:"ledgerId"`
    Sequence           int64           `json:"sequence"`
    Unit               string          `json:"unit"`
    Action             ChangeAction    `json:"action"`               // "PUT" | "DELETE"
    Desired            json.RawMessage `json:"desired,omitempty"`
    BaseFingerprint    string          `json:"baseFingerprint"`
    DesiredFingerprint string          `json:"desiredFingerprint"`
    IdempotencyKey     string          `json:"-"`                    // never serialised
    RequestFingerprint string          `json:"-"`                    // never serialised
    Status             ChangeStatus    `json:"status"`
    CreatedAt          time.Time       `json:"createdAt"`
}

type Proposal struct {
    ID            string         `json:"id"`
    LedgerID      string         `json:"ledgerId"`
    Title         string         `json:"title"`
    BaseReleaseID string         `json:"baseReleaseId,omitempty"`
    Hash          string         `json:"hash"`
    Status        ProposalStatus `json:"status"`
    ChangeIDs     []string       `json:"changeIds"`
    CreatedAt     time.Time      `json:"createdAt"`
}

type CheckResult struct {
    ID, ProposalID, ProposalHash, Kind, Summary string
    Passed    bool
    Evidence  []byte
    CreatedAt time.Time
}

type Approval struct {
    ID, ProposalID, ProposalHash, Actor string
    CreatedAt time.Time
}

type Release struct {
    ID           string    `json:"id"`
    LedgerID     string    `json:"ledgerId"`
    ProposalID   string    `json:"proposalId"`
    ProposalHash string    `json:"proposalHash"`
    ParentID     string    `json:"parentId,omitempty"`
    Hash         string    `json:"hash"`
    CreatedAt    time.Time `json:"createdAt"`
}

type ReleaseIntent struct {
    ID             string              `json:"id"`
    LedgerID       string              `json:"ledgerId"`
    ProposalID     string              `json:"proposalId"`
    ProposalHash   string              `json:"proposalHash"`
    ParentID       string              `json:"parentId,omitempty"`
    Status         ReleaseIntentStatus `json:"status"`
    Plan           []byte              `json:"plan"`
    CreatedAt      time.Time           `json:"createdAt"`
    Resolution     string              `json:"resolution,omitempty"`
    ResolutionNote string              `json:"resolutionNote,omitempty"`
    ResolvedAt     *time.Time          `json:"resolvedAt,omitempty"`
}

// engine read model: Plan is expanded instead of exposing the persisted []byte.
type ReleaseIntentOperation struct {
    targets.Operation
    HasBeforeImage bool `json:"hasBeforeImage"`
}

type ReleaseIntentPlan struct {
    Operations []ReleaseIntentOperation `json:"operations"`
}

type RetryReleaseIntentResult struct {
    Resolved   bool                           `json:"resolved"`
    Mismatches []targets.VerificationMismatch `json:"mismatches"`
}

type ActionGate struct {
    Enabled bool   `json:"enabled"`
    Reason  string `json:"reason"`
}

type ProposalGates struct {
    HasCurrentPassingCheck bool       `json:"hasCurrentPassingCheck"`
    HasCurrentApproval     bool       `json:"hasCurrentApproval"`
    BaseMatchesHead        bool       `json:"baseMatchesHead"`
    Releasable             bool       `json:"releasable"`
    Reason                 string     `json:"reason"`
    ApprovalAction         ActionGate `json:"approvalAction"`
    ReleaseAction          ActionGate `json:"releaseAction"`
}

type ProposalDetail struct {
    Proposal             ledger.Proposal `json:"proposal"`
    Changes              []ledger.Change `json:"changes"`
    CurrentHeadReleaseID string          `json:"currentHeadReleaseId"`
    Gates                ProposalGates   `json:"gates"`
}
```

The read-API `CheckResult` exposes `id`, `proposalHash`, `kind`, `passed`, `summary`, `createdAt`, and `current`. Valid stored evidence is emitted as decoded JSON under `evidence`; malformed or absent blobs omit it and set `evidenceUnavailable: true`. The read-API `Approval` exposes `id`, `proposalHash`, `actor`, `createdAt`, and `current`. `current` means the stored `proposalHash` equals the Proposal's current hash.

### Enumerations

| Type | Values |
|---|---|
| `ChangeAction` | `PUT`, `DELETE` |
| `ChangeStatus` | `ACCEPTED`, `READY`, `INVALID`, `RELEASED` |
| `ProposalStatus` | `DRAFT`, `REVIEWED`, `APPROVED`, `RELEASED`, `BLOCKED` |
| `ReleaseIntentStatus` | `READY`, `APPLYING`, `VERIFYING`, `FINALIZED`, `RECOVERY_REQUIRED`, `ABANDONED` |

`Proposal.status` is written as `DRAFT`, `REVIEWED` or `BLOCKED` after evaluation, `APPROVED` after approval, and `RELEASED` after finalization. These values summarize workflow activity; release authority is always recomputed from current evidence, approval, and HEAD predicates. `Change.status` is only ever written as `READY` or `RELEASED`.

---

## 5. Identity and hashing (`internal/ledger/identity.go`, `invariants.go`)

```go
func NewID(prefix string) (string, error)  // prefix + "_" + hex(crypto/rand 16 bytes)
func Hash(value any) (string, error)       // "sha256:" + hex(sha256(json.Marshal(value)))
func Fingerprint(value json.RawMessage) string // unmarshal→remarshal for canonical form, then sha256
```

ID prefixes in use: `ldg`, `chg`, `pr`, `apr`, `chk`, `intent`, `rel`.

Canonicalisation relies on Go's `encoding/json` map-key sorting and struct field order. **Any change to a hashed struct's field order or names changes every hash.** Treat hashed structs as versioned wire formats — both carry an explicit `"version": 1`.

```go
// Proposal identity — order-sensitive over ChangeIDs
ProposalHash = Hash(struct{ Version int; LedgerID, BaseReleaseID string; ChangeIDs []string }{1, …})

// Release identity — includes parent and the proposal hash
ReleaseHash  = Hash(struct{ Version int; LedgerID, ProposalID, ParentID, ProposalHash string }{1, …})

// Change idempotency identity
requestFingerprint = Hash(struct{ Unit string; Action ChangeAction; Desired json.RawMessage }{…})
```

Object store hashes are **different**: `sha256(kind || 0x00 || value)`, prefixed `sha256:`. Kinds in use: `VALUE`, `BEFORE_IMAGE`.

### Sentinel errors

`ErrInvalid`, `ErrConflict`, `ErrStaleEvidence`, `ErrReleaseNotReady` in `ledger`; `ErrNotFound`, `ErrIdempotencyConflict`, `ErrChangeClaimed` in `repository`.

---

## 6. Engine API

```go
func New(repo repository.Repository, target targets.TargetAdapter, provider inference.Provider) *Engine

func (e *Engine) CreateLedger(ctx, name, description string) (ledger.Ledger, error)
func (e *Engine) ListLedgers(ctx) ([]ledger.Ledger, error)
func (e *Engine) CreateChange(ctx, ledgerID string, r CreateChangeRequest) (ledger.Change, error)
func (e *Engine) ListChanges(ctx, ledgerID string) ([]ledger.Change, error)
func (e *Engine) CreateProposal(ctx, ledgerID string, r CreateProposalRequest) (ledger.Proposal, error)
func (e *Engine) ListProposals(ctx, ledgerID string) ([]ledger.Proposal, error)
func (e *Engine) LoadProposalDetail(ctx, ledgerID, proposalID string) (ProposalDetail, error)
func (e *Engine) ListCheckResults(ctx, ledgerID, proposalID string) ([]CheckResult, error)
func (e *Engine) ListApprovals(ctx, ledgerID, proposalID string) ([]Approval, error)
func (e *Engine) EvaluateProposal(ctx, ledgerID, proposalID, criteria string) (EvaluationResponse, error)
func (e *Engine) ApproveProposal(ctx, ledgerID, proposalID, actor string) error
func (e *Engine) ReleaseProposal(ctx, ledgerID, proposalID string) (ledger.Release, error)
func (e *Engine) ListReleaseIntents(ctx, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ReleaseIntent, error)
func (e *Engine) LoadReleaseIntent(ctx, ledgerID, intentID string) (ReleaseIntent, error)
func (e *Engine) RetryReleaseIntent(ctx, ledgerID, intentID string) (RetryReleaseIntentResult, error)
func (e *Engine) ResolveReleaseIntent(ctx, ledgerID, intentID, resolution, note string) error
func (e *Engine) ListReleases(ctx, ledgerID string) ([]ledger.Release, error)
func (e *Engine) CreateRollbackProposal(ctx, ledgerID, targetReleaseID string) (ledger.Proposal, error)
func (e *Engine) RecoverReleases(ctx) error
func (e *Engine) Events() *Broker
func (e *Engine) TargetCapabilities() targets.Capabilities
func (e *Engine) InferenceName() string   // "disabled" when provider is nil
func (e *Engine) Close() error
```

`Engine` holds a `releaseMu sync.Mutex`. `ReleaseProposal` and `RecoverReleases` both take it — release work is serialised process-wide.

### Business rules by method

**`CreateChange`** — see [product.md §3](product.md). Key details:
- `PUT` desired JSON is compacted via `json.Compact`; invalid JSON ⇒ `INVALID_ARGUMENT`.
- `DELETE` forces `Desired = nil`.
- Repository scans route nullable `desired` through `[]byte`, so lists containing DELETE Changes remain readable and omit `desired` from the JSON response.
- Idempotency is checked **before** insert and again on insert failure (race-safe fallback).
- Status is set to `READY` at insert.
- Desired bytes are written to the CAS as `VALUE` before the row insert.

**`CreateProposal`**
- `title` and at least one `changeId` required.
- Every loaded Change must be `READY`; duplicates rejected.
- `baseReleaseId = CurrentHead(ledgerID).ReleaseID`.
- Hash computed **after** `ChangeIDs` is copied; order is preserved from the request.
- `InsertProposal` failure is reported as `CONFLICT` ("already in another active Proposal") because the unique claim is the expected failure mode.

**`EvaluateProposal`**
- Calls `target.Preview` only — never `Apply`.
- Builds `{proposal, changes, preview}` as the evidence context.
- Inference disabled ⇒ `passed: true`, `kind: "deterministic"`, evidence = the context document.
- Inference enabled ⇒ provider result verbatim, `kind: "natural-language"`, evidence = marshalled result. Provider failure ⇒ `UNAVAILABLE`.

**`ApproveProposal`**
- Requires `HasPassingCheck(proposalID, proposal.Hash)`.
- Empty actor defaults to `local-user`.

**`LoadProposalDetail`**
- Loads the Proposal through `(ledgerID, proposalID)`, so a cross-Ledger ID is `NOT_FOUND`.
- Returns ordered Changes, current HEAD, and server-authoritative approval/release action gates.
- The release gate and `ReleaseProposal` share `evaluateGates`; missing passing evidence, missing approval, and moved HEAD therefore return identical reason strings.
- Gate booleans read current hash-bound evidence and approval rows; Proposal status is not a gate predicate.

**`ReleaseProposal`** — see [product.md §3 step 5](product.md).

Before gate evaluation, `ReleaseProposal` rejects a Ledger with any `RECOVERY_REQUIRED` Intent. `RetryReleaseIntent` accepts only `RECOVERY_REQUIRED` or `VERIFYING`, calls `TargetAdapter.Verify` without calling `Apply`, and shares `finalizeIntent` with the happy path. Structured verification mismatches return a successful HTTP response with `resolved: false`; transport or target-read failures remain `UNAVAILABLE`. `ResolveReleaseIntent` atomically records `ABANDONED`, its operator note, and its timestamp without changing `HEAD` or Proposal status.

**`CreateRollbackProposal`** — see [product.md §4](product.md). Note the synthetic idempotency key format: `rollback:{releases[0].ID}:{targetReleaseID}:{unit}` where `releases[0]` is the current HEAD release.

---

## 7. Repository

```go
type Repository interface {
    CreateLedger(context.Context, ledger.Ledger) error
    ListLedgers(context.Context) ([]ledger.Ledger, error)
    FindChangeByIdempotencyKey(ctx, ledgerID, key string) (ledger.Change, error)
    InsertChange(context.Context, *ledger.Change) error
    ListChanges(ctx, ledgerID string) ([]ledger.Change, error)
    LoadChanges(ctx, ledgerID string, ids []string) ([]ledger.Change, error)
    InsertProposal(context.Context, ledger.Proposal) error
    LoadProposal(ctx, ledgerID, proposalID string) (ledger.Proposal, error)
    ListProposals(ctx, ledgerID string) ([]ledger.Proposal, error)
    SaveCheckResult(context.Context, ledger.CheckResult) error
    ListCheckResults(ctx, proposalID string) ([]ledger.CheckResult, error)
    HasPassingCheck(ctx, proposalID, proposalHash string) (bool, error)
    SaveApproval(context.Context, ledger.Approval) error
    ListApprovals(ctx, proposalID string) ([]ledger.Approval, error)
    HasApproval(ctx, proposalID, proposalHash string) (bool, error)
    CurrentHead(ctx, ledgerID string) (ledger.Head, error)
    SaveReleaseIntent(context.Context, ledger.ReleaseIntent) error
    UpdateReleaseIntent(ctx, intentID string, s ledger.ReleaseIntentStatus) error
    ResolveReleaseIntent(ctx context.Context, intentID, note string, resolvedAt time.Time) error
    LoadReleaseIntent(ctx context.Context, intentID string) (ledger.ReleaseIntent, error)
    ListReleaseIntentsForLedger(ctx context.Context, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ledger.ReleaseIntent, error)
    ListUnfinishedReleaseIntents(context.Context) ([]ledger.ReleaseIntent, error)
    LoadReleaseIntentForProposal(ctx, proposalID string) (ledger.ReleaseIntent, error)
    FinalizeRelease(context.Context, ledger.ReleaseIntent, ledger.Release) error
    ListReleases(ctx, ledgerID string) ([]ledger.Release, error)
    WriteObject(ctx, kind string, value []byte) (hash string, err error)
    ReadObject(ctx, hash string) ([]byte, error)
    Close() error
}
```

Methods describe **Gyrifi operations**, not generic CRUD. Do not add a `Query(sql string)` style escape hatch.

### SQLite

Pragmas applied at open, in order:

```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=FULL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
```

Migrations are embedded with `//go:embed` in `runtime/migrations/migrations.go` and applied in filename order before anything else runs. Each applied version is recorded in `schema_migrations`.

`FinalizeRelease` is the only multi-table transaction: insert `releases`, update `ledger_heads`, set the Proposal `RELEASED`, set its Changes `RELEASED`, set the intent `FINALIZED` — with a HEAD-unchanged guard inside the transaction.

### Object store (`internal/repository/objects.go`)

```text
{ObjectsPath}/{hash[0:2]}/{hash[2:]}
```

- Hash: `sha256(kind || 0x00 || value)`; the returned handle is `"sha256:" + hex`.
- Existing path ⇒ returned immediately (deduplicated, idempotent).
- Write: `os.CreateTemp` in the **same directory** → `Chmod(0o640)` → `Write` → `Sync` → `Close` → `os.Rename`.
- Directories are `0o750`. Read validates a 64-char hex hash.
- Deliberately absent: pack files, remote object stores, deltas, repacking.

---

## 8. Schema (`runtime/migrations/001_initial.sql`)

```sql
CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);

CREATE TABLE ledgers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE ledger_heads (
    ledger_id TEXT PRIMARY KEY REFERENCES ledgers(id) ON DELETE CASCADE,
    release_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE changes (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    unit_key TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('PUT', 'DELETE')),
    desired BLOB,
    base_fingerprint TEXT NOT NULL DEFAULT '',
    desired_fingerprint TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (ledger_id, sequence),
    UNIQUE (ledger_id, idempotency_key)
);
CREATE INDEX changes_inbox        ON changes(ledger_id, status, sequence DESC);
CREATE INDEX changes_unit_history ON changes(ledger_id, unit_key, sequence DESC);

CREATE TABLE proposals (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    base_release_id TEXT NOT NULL DEFAULT '',
    proposal_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX proposals_by_ledger ON proposals(ledger_id, created_at DESC);

CREATE TABLE proposal_changes (
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    change_id TEXT NOT NULL REFERENCES changes(id),
    ordinal INTEGER NOT NULL,
    PRIMARY KEY (proposal_id, change_id),
    UNIQUE (change_id)                 -- ← invariant 4: one Proposal per Change
);

CREATE TABLE checks (
    id TEXT PRIMARY KEY,
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    proposal_hash TEXT NOT NULL,
    kind TEXT NOT NULL,                -- "deterministic" | "natural-language"
    passed INTEGER NOT NULL,
    summary TEXT NOT NULL,
    evidence BLOB,
    created_at TEXT NOT NULL
);
CREATE INDEX checks_current ON checks(proposal_id, proposal_hash, created_at DESC);

CREATE TABLE approvals (
    id TEXT PRIMARY KEY,
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    proposal_hash TEXT NOT NULL,
    actor TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (proposal_id, proposal_hash, actor)
);

CREATE TABLE release_intents (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id),
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    proposal_hash TEXT NOT NULL,
    parent_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    plan BLOB NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX unfinished_intents ON release_intents(status, created_at);

CREATE TABLE releases (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id),
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    parent_id TEXT NOT NULL DEFAULT '',
    release_hash TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);
CREATE INDEX releases_by_ledger ON releases(ledger_id, created_at DESC);
```

All timestamps are TEXT in UTC (`time.Now().UTC()`).

### Migration 002 — Release Intent resolution

`002_release_intent_resolution.sql` adds nullable `resolution`, `resolution_note`, and `resolved_at` TEXT columns to `release_intents`. `ABANDONED` rows store `resolution = "ABANDONED"`, the trimmed operator note, and an RFC 3339 timestamp. `ListUnfinishedReleaseIntents` treats both `FINALIZED` and `ABANDONED` as terminal. The migration uses `002` rather than the ticket's planned `003` because GRF-213 landed before GRF-212; migration numbers follow actual apply order.

**Migration rules:** never edit an applied migration. Add `00N_description.sql`. SQLite cannot drop or alter most constraints — prefer additive columns with defaults, or a create-copy-rename table rebuild inside one transaction.

---

## 9. Target adapter

```go
type Capabilities struct {
    AtomicApply      bool `json:"atomicApply"`
    ExactPreview     bool `json:"exactPreview"`
    ConditionalWrite bool `json:"conditionalWrite"`
    Batch            bool `json:"batch"`
    Restore          bool `json:"restore"`
}

type Value struct {
    Unit        string          `json:"unit"`
    Value       json.RawMessage `json:"value"`
    Fingerprint string          `json:"fingerprint"`
    Exists      bool            `json:"exists"`
}

type Preview struct { Fidelity, Summary string }

type Operation struct {
    Unit                string
    Action              ledger.ChangeAction
    Desired             json.RawMessage
    ExpectedFingerprint string   // captured by the Engine before Apply
    ExpectedExists      bool
    DesiredFingerprint  string
    TargetMetric        string   // e.g. "Cosine"
    BeforeObjectHash    string   // CAS handle for rollback material
    BeforeExists        bool
}

type Plan struct { Operations []Operation }

type VerificationMismatch struct {
    Unit     string `json:"unit"`
    Expected string `json:"expected"`
    Observed string `json:"observed"`
}

type VerificationError struct {
    Mismatches []VerificationMismatch
}

type TargetAdapter interface {
    Read(context.Context, string) (Value, error)
    Fingerprint(context.Context, string) (string, error)
    Preview(context.Context, []ledger.Change) (Preview, error)
    Compile(context.Context, []ledger.Change) (Plan, error)
    Apply(context.Context, Plan) error
    Verify(context.Context, Plan) error
    Restore(context.Context, Plan) error
    Capabilities() Capabilities
}
```

### Qdrant adapter (`internal/targets/qdrant/qdrant.go`)

- Constructed with `New(rawURL, collection, apiKey)`; rejects a URL without scheme or host, and an empty collection.
- HTTP client timeout **20s**; response bodies capped at **8 MiB**.
- Every request sets `Content-Type: application/json` and, when configured, `api-key: {GYRIFI_QDRANT_API_KEY}`.
- Endpoint template: `{baseURL}/collections/{urlPathEscape(collection)}{path}`.

| Operation | Call |
|---|---|
| Read | `GET /points/{unit}` — 404 ⇒ `Value{Exists:false}`, not an error |
| Collection metric | `GET ` (collection root) to read the distance metric |
| Apply `PUT` | `PUT /points?wait=true` |
| Apply `DELETE` | `POST /points/delete?wait=true` |

**Normalisation.** Both target reads and desired values are reduced to the canonical logical point `{id, vector, payload}` before fingerprinting. Extra Qdrant response fields never affect identity.

**Cosine collections.** Qdrant normalises vectors on write for `Cosine` distance, so a byte-identical fingerprint comparison would always fail. `Verify` compares vector **direction** with a dot-product tolerance of `1e-6` when the collection metric is `Cosine`; otherwise it compares strictly. Semantic disagreements are accumulated into `VerificationError.Mismatches`; read/network failures remain ordinary errors so recovery can distinguish a mismatch response from target unavailability. Only collections with a single unnamed vector config are supported — named multi-vector collections are rejected explicitly rather than verified with the wrong metric.

**Capabilities today:** `{AtomicApply: false, ExactPreview: false, ConditionalWrite: true, Batch: true, Restore: true}`. Sparse in-place release is advertised as **recoverable**, never as globally atomic. `Preview` fidelity is always `FAST`.

---

## 10. Inference

```go
type EvaluationRequest struct {
    ProposalHash string          `json:"proposalHash"`
    Context      json.RawMessage `json:"context"`
    Criteria     string          `json:"criteria"`
}

type Finding struct { Severity, Message, Unit string }

type EvaluationResult struct {
    Passed   bool      `json:"passed"`
    Summary  string    `json:"summary"`
    Findings []Finding `json:"findings"`
    Model    string    `json:"model"`
    Evidence any       `json:"evidence,omitempty"`
}

type Provider interface {
    Evaluate(context.Context, EvaluationRequest) (EvaluationResult, error)
    Name() string
}
```

`inference.StartLlamaServer(ctx, executable, modelPath, port)`:

- spawns `{executable} --host 127.0.0.1 --port {port} --model {modelPath}`,
- polls `GET http://127.0.0.1:{port}/health` every 250 ms with a 45 s deadline, accepting 2xx,
- returns a handle with `.Provider` and `.Stop()`.

Evaluation calls `POST /v1/chat/completions` with `temperature: 0` and `response_format: {"type":"json_object"}`. The prompt requires strict JSON with `passed`, `summary`, and `findings`. Unparsable or free-form output is **rejected as an error**, never coerced into a pass.

The provider has no access to `Repository` or `TargetAdapter`. Port 8081 is never exposed by the image.

---

## 11. Frontend contract

`studio/src/api/client.ts` exposes `request<T>`, `ApiError`, and the `api` object:

```ts
class ApiError extends Error {
    constructor(
        readonly code: string,
        message: string,
        readonly status: number,
        readonly kind: "transport" | "http",
        readonly requestId?: string,
    );
}

function request<T>(path: string, init?: RequestInit): Promise<T>;
function subscribeToRequestHealth(listener: (health: { reachable: true } | { reachable: false; error: ApiError }) => void): () => void;
```

`request` always sets `Content-Type: application/json`. A rejected `fetch` throws a `transport` `ApiError` with status `0` and publishes an unreachable request-health event; an intentional `AbortError` is passed through without changing reachability. Any HTTP response first publishes reachable, then a non-2xx response throws an `http` `ApiError` preserving `body.error.code`, `body.error.message`, and `X-Request-ID` when present. Malformed error envelopes use `code: "UNKNOWN"` and `Request failed ({status})`. A `204` resolves `undefined`. Read methods accept an optional `RequestInit` so callers can supply an `AbortSignal`. All paths are relative; Vite proxies `/api` and `/events` to `127.0.0.1:18080` in development, and production is same-origin.

`api` methods: `status`, `ledgers`, `createLedger`, `changes`, `createChange`, `proposals`, `createProposal`, `proposal`, `proposalChecks`, `proposalApprovals`, `evaluate`, `approve`, `release`, `releaseIntents`, `releaseIntent`, `retryReleaseIntent`, `resolveReleaseIntent`, `releases`, `rollback`. `evaluate` exposes the full persisted evidence payload (`passed`, `summary`, `previewFidelity`, optional `findings`, `model`, and `evidence`); `approve` sends the Studio's editable actor. The Release Intent methods provide the typed GRF-208 consumer contract, including expanded operations, before-image presence, and mismatch results.

`ProposalDetail.gates` contains aggregate release predicates plus `approvalAction` and `releaseAction` `{ enabled, reason }` values. `features/proposals/gates.ts` projects those per-action values by identity; feature code renders them verbatim and never derives governance permissions from Proposal status.

`studio/src/app/use-async.ts` owns the dependency-free query and mutation state primitives:

```ts
type QueryResult<T> = {
    data: T | undefined;
    error: Error | undefined;
    loading: boolean;
    refetching: boolean;
    unavailable: boolean;
    refetch: () => void;
};

function useQuery<T>(key: string, fn: (signal: AbortSignal) => Promise<T>, deps: unknown[]): QueryResult<T>;

type MutationResult<TArgs, TResult> = {
    run: (args: TArgs) => Promise<void>;
    pending: boolean;
    blocked: boolean;
    disabledReason: string | undefined;
    error: Error | undefined;
    result: TResult | undefined;
    reset: () => void;
};

function useMutation<TArgs, TResult>(fn: (args: TArgs) => Promise<TResult>): MutationResult<TArgs, TResult>;
```

`useQuery` aborts its active request on dependency change and unmount, does not update unmounted components, and preserves resolved data during a refetch while setting `refetching`. Transport failures set `unavailable` instead of a page error; resolved data remains visible and dimmed. `useMutation` guards duplicate invocations while pending, refuses to invoke while the shared reachability state is offline, returns the global banner message as `disabledReason`, and exposes HTTP failures for the caller to render. There is no cache or request deduplication beyond the active request lifecycle.

`ui/feedback/async-boundary.tsx` renders query loading, error/retry, empty, and populated states. It uses `Skeleton` while no data has resolved, `ErrorState` with `refetch` as Retry, defaults to arrays for empty detection, and retains populated content with the `gy-is-refetching` class (`opacity: 0.6`) during refetches.

`studio/src/api/events.ts` exposes the stateful `subscribeToEvents` lifecycle and the ledger-aware domain contract:

```ts
type DomainEvent = { kind: EventKind; ledgerId: string; subjectId: string; at: string };
function subscribeToLedgerEvents(ledgerId: string, handler: (event: DomainEvent) => void): EventSubscription;
```

Named SSE frames are parsed against the exact event-kind union; malformed or unknown frames are ignored. Native reconnect handles transient `CONNECTING` errors. A permanently `CLOSED` source is replaced explicitly with jittered exponential delays from 1 second to a 30-second cap and a six-attempt ceiling; after the ceiling, the topbar exposes `Reconnect`. Closing the subscription closes the source and clears its timer.

`app/reachability.tsx` owns the application-level Runtime and the one unfiltered stream subscription. It dispatches each domain event only to active invalidation callbacks registered for the matching Ledger; callbacks refetch REST state and never apply event payloads directly. A reconnect fans out to every callback because events may have been missed. The provider probes `api.status()` immediately, every 30 seconds while connected, and with a 1, 2, 4, 8, 16, 30-second retry sequence while offline. Polling and active probes stop while `document.hidden`; visibility restoration triggers an immediate retry. Any successful request clears the persistent transport banner. `features/shell/runtime-status.tsx` renders HTTP and stream state together; `app/reachability-banner.tsx` renders the offline message and immediate Retry action.

`app/error-boundary.tsx` is the class-based `ErrorBoundary({ fallback, onError?, children })`. The root boundary is inside `Providers` and renders a full-page `ErrorState`, error `CodeBlock`, and last failed request ID. The current routed page has a separately keyed section boundary, so navigation survives a page render error and reset remounts only that subtree. The boundary owns once-per-error logging; React's root `onCaughtError` default logger is disabled to avoid duplicate console entries.

State composition: `Providers` nests reachability around the AppState context `{ ledgerId, ledger, ledgers, setLedgerId, refreshLedgers, openLedgerSwitcher, ledgerSwitcherRequest }`; `openLedgerSwitcher()` increments the request token so ledger-scoped empty states can focus and open the shared topbar switcher. Its ledger list read uses `useQuery`, and only `ledgerId` is persisted to `localStorage["gyrifi.ledger"]`. The four feature pages use `useQuery`, `AsyncBoundary`, `useMutation`, and reconnect invalidation; mutation errors are rendered through `ErrorState`, and every mutation control consumes `blocked` and `disabledReason`. Routing is hash-based (`#ledgers`, `#changes`, `#proposals`, `#proposals/{proposalId}`, `#releases`), defaulting to `ledgers`. The structured route parser returns `{ area, id? }`, preserving Proposal detail selection across reloads.

The Releases page loads Releases, Proposals, and Release Intents as one workspace. It joins finalized Intents to Releases by Proposal ID to display plans and source titles, computes rollback impact as the unique units touched by all newer Release plans, and disables rollback rather than guessing when a plan is unavailable. Recovery actions call the GRF-213 endpoints, render semantic mismatches or server errors inline, and refetch REST state after success; SSE remains an invalidation hint only.

---

## 12. Test surface

| Location | Covers |
|---|---|
| `runtime/internal/engine/events_test.go` | broker delivery, full-buffer drops, idempotent unsubscribe, and concurrent publish/unsubscribe safety |
| `runtime/internal/interfaces/http/events_test.go` | SSE connected frame, domain forwarding, Ledger filtering, flushing, and context cancellation |
| `runtime/tests/change_flow_test.go` | full Change → Proposal → Evaluation → Approval → Release and event sequence; idempotent re-submission; rollback; apply/verify failure and recovery events |
| `runtime/internal/repository/sqlite_test.go` | CheckResult and Approval newest-first ordering, non-null empty lists, and nullable DELETE desired scanning/serialization |
| `runtime/tests/proposal_detail_test.go` | detail and action gates, release/approval reason anti-drift, moved HEAD, stale/malformed evidence, cross-Ledger isolation, and HTTP serialization |
| `runtime/tests/release_recovery_test.go` | Intent list/detail and scoping; before-image presence; retry success, mismatch, and unavailability; explicit abandonment; release guard and repeated-call behavior |
| `runtime/internal/ledger/invariants_test.go` | Proposal hash determinism and order sensitivity; approval staleness after re-hash |
| `runtime/internal/targets/qdrant/qdrant_test.go` | collection path construction, `api-key` header, cosine vector equivalence, and structured verification mismatches |
| `runtime/internal/inference/llamacpp_test.go` | structured-output requirement, free-form rejection |
| `studio/src/api/client.test.ts` | versioned endpoint paths including Proposal detail/evidence and Release Intent recovery, structured/request-ID error mapping, transport versus HTTP reachability |
| `studio/src/api/events.test.ts` | stream state, bounded CLOSED retry, manual reconnect, timer/source teardown, typed named-event parsing and dispatch |
| `studio/src/app/error-boundary.test.tsx` | fallback/reset contract and once-per-error logging |
| `studio/src/app/reachability.test.ts` | exact bounded reachability backoff schedule |
| `studio/src/features/changes/*.test.ts(x)` | Changes inbox states, eligibility, selection-bar contract, filtering, JSON validation, DELETE omission, ordering, and conflict placement |
| `studio/src/features/proposals/*.test.ts(x)` | Proposal route/list states, server-gate projection, progress and stale evidence/approval, ordered creation, confirmation, and HTTP-503 recovery guidance |
| `studio/src/features/releases/*.test.tsx` | timeline order and HEAD marking, five page states, plan/before-image rendering, unique rollback unit counts and forward-history copy, verbatim rollback errors, recovery presence and actions |
| `studio/src/features/shared/time.test.ts` | relative minute/hour/day age formatting and malformed/future timestamps |
| `studio/src/test/` | **empty** (GRF-230) |
| `e2e/` | **empty** (GRF-232) |

Large model downloads are never part of tests.

---

## 13. Quality gate

Every change must pass:

```sh
cd runtime
go fmt ./... && go vet ./... && go test ./... && go build ./cmd/gyrifi

cd ..
pnpm install --frozen-lockfile
pnpm typecheck && pnpm test && pnpm build

docker build -t gyrifi:dev .
```

`docker build` runs `go test ./...` inside the `runtime-build` stage, so a broken test breaks the image.

---

## 14. Known technical gaps

| Gap | Ticket |
|---|---|
| No Proposal cancellation, so claims are permanent | GRF-212 |
| List endpoints have no pagination, filtering, or bounds | GRF-214 |
| No authentication or authorisation anywhere | GRF-220 |
| `Change.baseFingerprint` is always `""`; no async preparation phase | GRF-221 |
| No retention budget, quota, or backup command | GRF-222 |
| `cli version` prints `gyrifi dev` while the API reports `0.1.0` | GRF-223 |
| No CI pipeline | GRF-233 |
| Qdrant adapter is only tested against a fake, never a live instance | GRF-231 |
| No `DELETE`/withdraw/archive route for any entity; `routes()` registers none | GRF-215 |
| No `/healthz` or `/readyz`; no metrics of any kind | GRF-224 |
| `llama-server` is never `Wait()`ed on and its `Stdout`/`Stderr` are `nil` | GRF-225 |
| No rate limiting; `SetMaxOpenConns(1)` makes write floods starve reads | GRF-226 |
