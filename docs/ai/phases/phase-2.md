# Phase 2 — Governance API completeness

**Goal:** close the gaps where the runtime holds state it will not let anyone read, or reaches a state it will not let anyone leave. Every one of these tickets fixes a place where the audit trail exists in the database but not in the API.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-210](../tickets/GRF-210-event-stream.md) | Real domain event stream | M | — | Done |
| [GRF-211](../tickets/GRF-211-proposal-detail-api.md) | Proposal detail and evidence read API | M | — | Done |
| [GRF-212](../tickets/GRF-212-proposal-cancellation.md) | Proposal cancellation and claim release | M | — | Not started |
| [GRF-213](../tickets/GRF-213-release-intent-api.md) | Release intent inspection and recovery API | L | — | Done |
| [GRF-214](../tickets/GRF-214-pagination.md) | Pagination and filtering for list endpoints | M | — | Not started |
| [GRF-215](../tickets/GRF-215-lifecycle-management.md) | Ledger and Change lifecycle management | M | GRF-212 | Not started |

## Phase-level notes

- **GRF-211 and GRF-213 are prerequisites for Phase 1 work** (GRF-207 and GRF-208 respectively). Schedule them accordingly — the recommended order in [INDEX.md](../tickets/INDEX.md) interleaves the phases for this reason.
- This phase introduces the first migrations after `001_initial.sql`. `001_initial.sql` is frozen; every schema change is a new numbered file. Migration numbers are allocated in ticket order: 002 (GRF-212), 003 (GRF-213), 004 (GRF-214), 007 (GRF-215). If tickets land out of order, renumber to match the actual apply order and record it here.
- GRF-213 landed before GRF-212, so its migration uses the next actual apply-order number, `002_release_intent_resolution.sql`, instead of the ticket's planned `003`.
- Three tickets add enum members — `ProposalStatus.CANCELLED` (GRF-212), `ReleaseIntentStatus.ABANDONED` (GRF-213), and `ChangeStatus.WITHDRAWN` (GRF-215). All must be reflected in `design-system.md` §2.2 status tone mapping and in the exhaustive TypeScript mapping, or the frontend build breaks. That breakage is intentional and desirable.
- GRF-212 and GRF-215 both manipulate `proposal_changes` claims. Do GRF-212 first; GRF-215's withdrawal guard depends on cancellation existing as the escape hatch for a claimed Change.
- No new Go dependencies are permitted in this phase.

## Exit criteria

- [ ] All six tickets complete.
- [ ] No state the runtime records is unreadable through the API.
- [ ] No state the runtime can enter is unresolvable through the API.
- [ ] No mistake a user can make is irreversible except by design.
- [ ] Every list endpoint is bounded.
- [ ] `/events/v1` carries real domain events and Studio no longer polls.
- [ ] `go test ./...` green, including new migration tests.

## Completed entries

### GRF-210 — Real domain event stream

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | No |

**What was built**

The Engine now owns a concurrency-safe in-process broker and publishes advisory events after every user-visible durable transition in the Change → Proposal → Release workflow. `/events/v1` fans those events out as named SSE frames with optional Ledger filtering, per-subscriber buffering, keepalives, and clean cancellation. Studio retains one shared EventSource, validates typed event payloads, and refetches matching REST-backed surfaces rather than treating stream data as authoritative state.

**Files added**

- `runtime/internal/engine/events.go` — event vocabulary, payload, non-blocking broker, dropped counter, and Engine accessor.
- `runtime/internal/engine/events_test.go` — delivery, buffer saturation, idempotent unsubscribe, and concurrent publish/unsubscribe tests.
- `runtime/internal/interfaces/http/events_test.go` — connected frame, Ledger filter, named frame, flush, cancellation, and unsubscribe coverage.

**Files changed**

- `runtime/internal/engine/engine.go` — Engine-owned broker construction.
- `runtime/internal/engine/changes.go` — publish only after a newly inserted Change; cached/racing idempotent returns remain silent.
- `runtime/internal/engine/proposals.go` — publish after Proposal and approval persistence.
- `runtime/internal/engine/evaluation.go` — publish after evidence persistence.
- `runtime/internal/engine/releases.go` — started/completed/failed/recovery-required events at durable boundaries.
- `runtime/internal/interfaces/http/server.go` — buffered broker subscription, optional `ledgerId` filter, domain frame writes, and write-failure exits.
- `runtime/tests/change_flow_test.go` — exact happy-path sequence, apply/verify failures, idempotent replay, and startup recovery events.
- `studio/src/api/types.ts`, `studio/src/api/events.ts`, `studio/src/api/events.test.ts` — typed named-event contract, parser, subscription, and tests.
- `studio/src/app/reachability.tsx`, `studio/src/app/use-ledger-events.ts` — Ledger-scoped invalidation over the existing shared stream.
- `studio/src/features/{ledgers,changes,proposals,releases}/*-page.tsx` — register each REST query for the relevant Ledger; Ledgers card counts register independently.
- `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/tickets/INDEX.md` — closed gaps, current tree/contracts, and status.

**Files removed**

None.

**Contracts introduced or changed**

```go
type EventKind string

const (
		EventChangeAccepted         EventKind = "change.accepted"
		EventProposalCreated        EventKind = "proposal.created"
		EventProposalEvaluated      EventKind = "proposal.evaluated"
		EventProposalApproved       EventKind = "proposal.approved"
		EventReleaseStarted         EventKind = "release.started"
		EventReleaseCompleted       EventKind = "release.completed"
		EventReleaseFailed          EventKind = "release.failed"
		EventIntentRecoveryRequired EventKind = "intent.recovery_required"
)

type Event struct {
		Kind      EventKind `json:"kind"`
		LedgerID  string    `json:"ledgerId"`
		SubjectID string    `json:"subjectId"`
		At        time.Time `json:"at"`
}

func (b *Broker) Subscribe(buffer int) (<-chan Event, func())
func (b *Broker) Publish(Event)
func (b *Broker) Dropped() uint64
func (e *Engine) Events() *Broker
```

```ts
type EventKind = "change.accepted" | "proposal.created" | "proposal.evaluated" |
	"proposal.approved" | "release.started" | "release.completed" |
	"release.failed" | "intent.recovery_required";
type DomainEvent = { kind: EventKind; ledgerId: string; subjectId: string; at: string };

function parseDomainEvent(data: string): DomainEvent | undefined;
function subscribeToLedgerEvents(
	ledgerId: string,
	handler: (event: DomainEvent) => void,
): EventSubscription;
function useLedgerEvents(ledgerId: string | undefined, onInvalidate: () => void): void;
```

SSE remains `GET /events/v1`; `?ledgerId=ldg_…` is optional. The initial `ledger` connected frame and 20-second keepalive are unchanged. Domain frames use `event: <kind>` and the JSON `Event` as `data`.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Hold the broker read lock through non-blocking sends | Unsubscribe takes the write lock before delete/close, eliminating send-on-closed-channel races | Copy channels under lock then send after unlock | A concurrent unsubscribe could close a copied channel before send |
| Give each SSE client a 16-event buffer | A brief slow writer does not immediately lose hints; saturation still cannot block the Engine | Unbuffered subscription | Every event would be dropped unless the handler happened to be receiving at that instant |
| Publish only after successful durable writes | REST remains authoritative and no client is prompted to refetch state that did not commit | Publish before repository calls | Failed governance operations would create false observable transitions |
| Use Intent ID for `release.started` and `release.failed` | No Release exists until finalization; recovery operates on the Intent | Generate/publish a future Release ID | It would expose an identity for a record that may never exist |
| Retain one unfiltered Studio stream and filter invalidations in the provider | Preserves GRF-209 lifecycle/reconnect ownership while supporting every active Ledger surface | Create an EventSource in each page or card | It multiplies reconnect loops and connections and violates the established shared-stream contract |
| Ignore malformed/unknown Studio frames | Stream data is advisory and must not break current REST-backed UI | Throw from the EventSource callback | One incompatible frame could stop live refresh behavior |

**Deviations from the ticket**

None. The broker exposes its dropped counter but does not log every increment. This follows the required non-blocking counter behavior and avoids noisy per-drop logging; future debug metrics may sample `Dropped()` without changing request paths.

**Traps for future work**

- Never close a broker subscriber outside the same mutex used by `Publish`; copying subscriber channels and releasing the lock first reintroduces a send/close panic.
- A Release ID does not exist at `release.started`. Consumers must interpret started/failed `subjectId` values as Intent IDs and completed values as Release IDs.
- The event stream is not an audit log. It has deliberate per-subscriber loss, no persistence, and no `Last-Event-ID`; every handler must refetch the authoritative REST resource.
- `CreateRollbackProposal` composes existing `CreateChange` and `CreateProposal` methods, so it naturally emits one event per newly synthesized Change plus `proposal.created`. Do not add duplicate rollback-specific publications.
- Register named EventSource listeners for every exact event kind; `onmessage` does not receive named SSE frames.

**Tests added**

- `runtime/internal/engine/events_test.go` — delivery, 100 full-buffer drops without blocking, dropped count, double unsubscribe, and publish/unsubscribe race safety.
- `runtime/internal/interfaces/http/events_test.go` — backwards-compatible handshake, headers, optional Ledger filter, named JSON frame, flush, request cancellation, and post-unsubscribe publish safety.
- `runtime/tests/change_flow_test.go` additions — exact six-event full-flow order and subjects; idempotent replay silence; apply and verify `release.failed`; recovery-required and recovered completion behavior.
- `studio/src/api/events.test.ts` additions — URL encoding, named-listener dispatch, typed parsing, and malformed/unknown payload rejection.

**Docs updated**

- `docs/ai/tech-spec.md` §3 / §6 / §11 / §12 / §14 — SSE table, Engine and Studio signatures, test coverage, and closed gap.
- `docs/ai/product.md` §7 — removed the keepalive-only product gap.
- `docs/ai/repo-structure.md` §2 — broker and SSE test files.
- `docs/ai/tickets/INDEX.md` — GRF-210 marked Done.
- `docs/ai/phases/phase-2.md` — phase status, ticket table, and this completion entry.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine           (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference        (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http  (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger           (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant   (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                     1.756s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  8 passed (8)
Tests       25 passed (25)
✓ 1868 modules transformed.
✓ built in 1.06s

$ docker build -t gyrifi:local .
[+] Building 34.1s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

**Follow-ups discovered**

- A sampled debug metric can expose `Broker.Dropped()` if operators need slow-consumer visibility; no per-event log belongs on the governance request path.
- A future cross-process or replay requirement needs a different architecture and an ADR. This broker intentionally supports only the product's single-replica, refetch-hint contract.

### GRF-211 — Proposal detail and evidence read API

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — owner-approved per-action gate extension; stale status documentation corrected to match source |

**What was built**

Three Ledger-scoped read endpoints now return one Proposal with its ordered Changes and authoritative gates, all stored evaluation evidence, and all approvals. The Engine computes aggregate release readiness and separate approval/release action gates from current hash-bound evidence, approval, and HEAD state. The mutation methods share those predicates and exact disabled reasons, so Studio can render the server response without recreating governance logic. Valid evidence is emitted as JSON; malformed blobs remain readable metadata with `evidenceUnavailable: true`.

**Files added**

- `runtime/internal/engine/proposal_detail.go` — detail response types, approval/release action gates, shared predicates, evidence conversion, and scoped read methods.
- `runtime/internal/repository/sqlite_test.go` — newest-first checks/approvals and non-null empty-list tests.
- `runtime/tests/proposal_detail_test.go` — gate anti-drift, moved HEAD, evidence, scoping, secret-field, and HTTP response tests.

**Files changed**

- `runtime/internal/repository/repository.go`, `runtime/internal/repository/sqlite.go` — stored CheckResult and Approval list contracts and SQL.
- `runtime/internal/engine/proposals.go`, `runtime/internal/engine/releases.go` — shared approval and release gate evaluation; granular release-disabled messages.
- `runtime/internal/interfaces/http/server.go` — three GET routes and thin handlers, including the required GRF-220 evidence-authorisation note.
- `studio/src/api/types.ts`, `studio/src/api/client.ts`, `studio/src/api/client.test.ts` — response types, methods, and exact path coverage.
- `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/tickets/INDEX.md`, `docs/ai/phases/phase-2.md` — current behavior, contracts, tree, and status.

**Files removed**

None.

**Contracts introduced or changed**

```go
func (repository *SQLite) ListCheckResults(ctx context.Context, proposalID string) ([]ledger.CheckResult, error)
func (repository *SQLite) ListApprovals(ctx context.Context, proposalID string) ([]ledger.Approval, error)

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

func (e *Engine) LoadProposalDetail(ctx context.Context, ledgerID, proposalID string) (ProposalDetail, error)
func (e *Engine) ListCheckResults(ctx context.Context, ledgerID, proposalID string) ([]CheckResult, error)
func (e *Engine) ListApprovals(ctx context.Context, ledgerID, proposalID string) ([]Approval, error)
```

```ts
type ActionGate = { enabled: boolean; reason: string };
type ProposalGates = {
	hasCurrentPassingCheck: boolean;
	hasCurrentApproval: boolean;
	baseMatchesHead: boolean;
	releasable: boolean;
	reason: string;
	approvalAction: ActionGate;
	releaseAction: ActionGate;
};

proposal: (ledgerId: string, proposalId: string, init?: RequestInit) => request<ProposalDetail>(...)
proposalChecks: (ledgerId: string, proposalId: string, init?: RequestInit) => request<{ items: CheckResult[] }>(...)
proposalApprovals: (ledgerId: string, proposalId: string, init?: RequestInit) => request<{ items: Approval[] }>(...)
```

The endpoints are:

- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}`
- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/checks`
- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals`

All three first load the Proposal by both Ledger and Proposal ID. A Proposal owned by another Ledger therefore returns `404 NOT_FOUND` without exposing its existence or dependent rows.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Return server-computed `approvalAction` and `releaseAction` gates in addition to the ticket's aggregate fields | Owner-approved prerequisite for GRF-207; each button receives its own authoritative reason | Let Studio infer approval from `hasCurrentPassingCheck` or Proposal status | Browser governance logic can drift and is forbidden by the architecture |
| Share `evaluateApprovalGate` with `ApproveProposal`, and `evaluateGates` with `ReleaseProposal` | The read and mutation paths use the same predicates and message constants | Duplicate conditions in the read method | Tests might catch current drift but could not prevent later divergence |
| Use granular release reasons with evaluation before approval before moved HEAD precedence | The response can explain the next blocking condition and the mutation returns the identical text | Keep the former combined evidence-and-approval error | It cannot distinguish the ticket's required pre-evaluation and post-evaluation states |
| Scope in the Engine by loading `(ledgerID, proposalID)` before repository evidence reads | Repository list signatures intentionally accept only Proposal ID, while the API must prevent cross-Ledger reads | Add Ledger ID to ticket-specified Repository methods | That would violate the explicit repository contract without improving the unique Proposal-ID query |
| Decode evidence into `json.RawMessage` only after `json.Valid` | Preserves arbitrary JSON shapes without remarshal or base64 encoding | Return ledger `[]byte` directly | `encoding/json` would expose base64 instead of the stored evidence document |
| Order by `created_at DESC, id DESC` | Meets newest-first and gives deterministic ordering for timestamp ties | Add an approvals index/migration | The ticket expressly forbids schema changes; the current dataset and query do not require one |

**Deviations from the ticket**

- Per owner decision, `ProposalGates` extends the specified five aggregate fields with `approvalAction` and `releaseAction` `{ enabled, reason }` values. This is additive and required so GRF-207 never derives whether either mutation is allowed.
- The ticket note says `REVIEWED`/`APPROVED` remain unused, but current `SaveCheckResult` writes `REVIEWED` or `BLOCKED` and `SaveApproval` writes `APPROVED`. Ground-truth precedence makes source authoritative. This ticket did not add, remove, or change those transitions; it corrected `product.md` and `tech-spec.md` to describe them and kept gates independent of status.
- No acceptance criterion was omitted. No schema change or dependency was added.

**Traps for future work**

- Do not serialize `ledger.CheckResult` directly from the read endpoint: its `[]byte` evidence becomes base64. Keep the Engine read model and malformed-evidence fallback.
- Proposal status is a display summary. It can be overwritten by later evaluations and must never replace current hash-bound evidence/approval checks.
- Gate reason precedence is observable API behavior: missing current passing evaluation, then missing current approval, then moved HEAD.
- Approval and release controls in GRF-207 must consume `approvalAction` and `releaseAction` verbatim; even deriving `enabled` from the aggregate booleans duplicates a governance decision.
- The evidence handler carries `TODO(GRF-220)` because evidence reads must receive the same authorisation as Change reads when authentication lands.

**Tests added**

- Repository tests persist out-of-order checks and approvals, assert newest-first deterministic reads, and verify empty results serialize from non-nil slices.
- Proposal detail integration tests cover pre-evaluation, evaluated, approved, and moved-HEAD gates; exact parity with `ApproveProposal`/`ReleaseProposal` errors; stale current flags; valid and malformed evidence; newest-first approvals; cross-Ledger Engine and HTTP `NOT_FOUND`; decoded JSON; and continued omission of Change idempotency fields.
- Studio client tests assert all three exact endpoint paths.

**Docs updated**

- `docs/ai/tech-spec.md` §3 / §4 / §6 / §7 / §11 / §12 / §14 — endpoints, read types, Engine/Repository/client contracts, tests, and closed gap.
- `docs/ai/product.md` §2 / §7 — source-accurate Proposal status behavior and closed detail gap.
- `docs/ai/repo-structure.md` §2 — new Engine and test files.
- `docs/ai/tickets/INDEX.md` — GRF-211 marked Done.
- `docs/ai/phases/phase-2.md` — ticket table and this record.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine           1.640s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http 2.043s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository       2.611s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                     3.388s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  8 passed (8)
Tests       26 passed (26)
✓ 1868 modules transformed.
✓ built in 1.08s

$ docker build -t gyrifi:local .
[+] Building 33.1s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local
```

**Follow-ups discovered**

None beyond the ticketed GRF-220 authorisation requirement and GRF-207 consumer.

### GRF-213 — Release intent inspection and recovery API

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — migration renumbered from planned 003 to actual apply-order 002 because GRF-213 landed before GRF-212 |

**What was built**

Four Ledger-scoped endpoints expose Release Intents and their expanded Plans, including a live object-store presence check for each retained before-image. Operators can retry verification without reapplying target writes or explicitly abandon a recovery attempt with a required note. Structured target mismatches are distinct from target unavailability, successful retry shares the ordinary atomic finalisation path, and unresolved recovery blocks new releases on the Ledger. The Engine emits `intent.resolved` after either successful retry or abandonment.

**Files added**

- `runtime/internal/engine/release_intents.go` — Intent read models, scoped reads, status validation, verification-only retry, mismatch responses, and explicit resolution.
- `runtime/migrations/002_release_intent_resolution.sql` — additive resolution, note, and timestamp columns.
- `runtime/tests/release_recovery_test.go` — Intent API, retry, resolution, release guard, before-image, scoping, and repeated-call coverage.

**Files changed**

- `runtime/internal/ledger/release.go` — `ABANDONED` plus persisted resolution metadata.
- `runtime/internal/targets/target.go`, `runtime/internal/targets/qdrant/qdrant.go` — structured semantic verification mismatch contract while preserving cosine-aware equivalence.
- `runtime/internal/repository/repository.go`, `runtime/internal/repository/sqlite.go` — load/list/resolve operations, nullable field scans, and terminal abandonment handling.
- `runtime/internal/engine/releases.go`, `runtime/internal/engine/events.go` — unresolved-recovery guard, shared finalisation, durable recovery events, and `intent.resolved`.
- `runtime/internal/interfaces/http/server.go` — two read and two recovery routes with thin request/response translation.
- `runtime/internal/repository/sqlite_test.go`, `runtime/internal/targets/qdrant/qdrant_test.go`, `runtime/tests/change_flow_test.go` — repository transitions, adapter mismatch structure, no-reapply instrumentation, and target-read failure behavior.
- `studio/src/api/types.ts`, `studio/src/api/client.ts`, `studio/src/api/client.test.ts` — typed Release Intent contracts and all four client methods.
- `studio/src/api/events.ts`, `studio/src/api/events.test.ts` — `intent.resolved` event support.
- `studio/src/features/shared/status.ts`, `studio/src/features/shared/status.test.ts` — exhaustive neutral `ABANDONED` mapping.
- `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/design-system.md`, `docs/ai/tickets/GRF-213-release-intent-api.md`, `docs/ai/tickets/INDEX.md` — workflow, invariants, contracts, tree, migration, status tone, acceptance, and completion status.

**Files removed**

None.

**Contracts introduced or changed**

```go
const IntentAbandoned ReleaseIntentStatus = "ABANDONED"

type VerificationMismatch struct {
	Unit     string `json:"unit"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type VerificationError struct {
	Mismatches []VerificationMismatch
}

type RetryReleaseIntentResult struct {
	Resolved   bool                           `json:"resolved"`
	Mismatches []targets.VerificationMismatch `json:"mismatches"`
}

func (e *Engine) ListReleaseIntents(ctx context.Context, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ReleaseIntent, error)
func (e *Engine) LoadReleaseIntent(ctx context.Context, ledgerID, intentID string) (ReleaseIntent, error)
func (e *Engine) RetryReleaseIntent(ctx context.Context, ledgerID, intentID string) (RetryReleaseIntentResult, error)
func (e *Engine) ResolveReleaseIntent(ctx context.Context, ledgerID, intentID, resolution, note string) error

func (r *SQLite) ResolveReleaseIntent(ctx context.Context, id, note string, resolvedAt time.Time) error
func (r *SQLite) LoadReleaseIntent(ctx context.Context, id string) (ledger.ReleaseIntent, error)
func (r *SQLite) ListReleaseIntentsForLedger(ctx context.Context, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ledger.ReleaseIntent, error)
```

```ts
type ReleaseIntentStatus = "READY" | "APPLYING" | "VERIFYING" | "FINALIZED" | "RECOVERY_REQUIRED" | "ABANDONED";
type RetryReleaseIntentResult = {
  resolved: boolean;
  mismatches: Array<{ unit: string; expected: string; observed: string }>;
};
```

HTTP routes:

- `GET /api/v1/ledgers/{ledgerID}/release-intents?status=<optional status>`
- `GET /api/v1/ledgers/{ledgerID}/release-intents/{intentID}`
- `POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/retry`
- `POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/resolve`

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Retry calls `Verify` only and uses the shared `finalizeIntent` tail on success | A partial apply cannot safely be repeated; one finalisation path prevents audit-history drift | Call `Apply` again or duplicate finalisation in retry | Reapplying may overwrite foreign state; duplicate transactional logic will diverge |
| Target adapters return `VerificationError` only for semantic disagreement | The API must return mismatch details with HTTP 200 while keeping transport/read failures as 503 | Parse adapter error strings or classify every verification error as a mismatch | Strings are not a contract; connectivity failures contain no trustworthy observed state |
| Derive `hasBeforeImage` by reading the object store | The object store, not a stale database flag, is authoritative for rollback material | Persist a presence boolean | It could outlive or disagree with the actual object |
| Resolve uses one conditional SQL update from `RECOVERY_REQUIRED` | The status check and resolution metadata must be atomic | Load, check, then perform an unconditional update | Another caller could resolve or finalize between the check and write |
| Publish one `intent.resolved` event for finalization and abandonment | Both transitions close operator recovery and require the same REST refetch | Add separate success/abandon event kinds | Consumers need invalidation, not event-derived state |

**Deviations from the ticket**

- The migration is `002_release_intent_resolution.sql`, not the planned `003_release_intent_resolution.sql`. GRF-213 landed before GRF-212, and Phase 2 requires numbering by actual apply order. The schema content is unchanged from the acceptance criterion.
- No behavior or acceptance criterion was omitted.

**Traps for future work**

- A `VerificationError` means verification completed and found semantic mismatches. Any ordinary adapter error means verification could not complete and must remain `UNAVAILABLE`; do not collapse these cases.
- Retry must never call `Apply` or `Restore`. Its only target mutation-adjacent call is `Verify`, followed by SQLite finalisation when verified.
- `hasBeforeImage` is intentionally evaluated at read time. `os.ErrNotExist` produces `false`; other object-store failures are internal errors, not missing-object claims.
- `ABANDONED` is terminal for startup recovery but leaves the Proposal and Changes untouched, allowing a later release after manual target repair.
- Resolution is intentionally not idempotent: resolving an already terminal Intent returns `409` rather than rewriting operator history.

**Tests added**

- Retry success finalizes through the shared path, advances `HEAD` once, emits completion/resolution events, and never increments target apply calls; a repeated retry conflicts.
- Semantic mismatch returns unit-level expected/observed fingerprints; target read failure is unavailable and leaves recovery state unchanged; a `VERIFYING` mismatch becomes `RECOVERY_REQUIRED`.
- Resolution validates the enum and trimmed note, records metadata, preserves Proposal/HEAD state, blocks repeated resolution, unblocks later release, and enforces the Ledger-wide release guard.
- List/detail endpoints cover newest-first status filtering, invalid status, cross-Ledger isolation, expanded Plan JSON, and present/missing before-images.
- SQLite tests cover filtered listing, atomic abandonment, repeated-resolution conflict, and exclusion of abandoned rows from startup recovery.
- Qdrant tests cover structured semantic mismatches while retaining cosine-aware comparison; Studio tests cover endpoint paths, event parsing, and exhaustive status tones.

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race -count=1 && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine 1.626s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference 2.036s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http 2.466s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger 2.857s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository 4.048s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 3.350s
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 4.975s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files 17 passed (17)
Tests 65 passed (65)
✓ 1865 modules transformed.
✓ built in 905ms

$ docker build -t gyrifi:local .
[+] Building 37.5s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ diff <ticket files> <INDEX status rows>
tickets consistent

$ git diff --check
(no output)
```

**Follow-ups discovered**

- GRF-208 should render the API's `hasBeforeImage` and mismatch values directly and use `intent.resolved` only as a refetch hint.
- GRF-222 should define retention for before-image objects referenced by abandoned intents.

