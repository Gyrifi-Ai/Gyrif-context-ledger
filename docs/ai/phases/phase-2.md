# Phase 2 — Governance API completeness

**Goal:** close the gaps where the runtime holds state it will not let anyone read, or reaches a state it will not let anyone leave. Every one of these tickets fixes a place where the audit trail exists in the database but not in the API.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-210](../tickets/GRF-210-event-stream.md) | Real domain event stream | M | — | Done |
| [GRF-211](../tickets/GRF-211-proposal-detail-api.md) | Proposal detail and evidence read API | M | — | Not started |
| [GRF-212](../tickets/GRF-212-proposal-cancellation.md) | Proposal cancellation and claim release | M | — | Not started |
| [GRF-213](../tickets/GRF-213-release-intent-api.md) | Release intent inspection and recovery API | L | — | Not started |
| [GRF-214](../tickets/GRF-214-pagination.md) | Pagination and filtering for list endpoints | M | — | Not started |
| [GRF-215](../tickets/GRF-215-lifecycle-management.md) | Ledger and Change lifecycle management | M | GRF-212 | Not started |

## Phase-level notes

- **GRF-211 and GRF-213 are prerequisites for Phase 1 work** (GRF-207 and GRF-208 respectively). Schedule them accordingly — the recommended order in [INDEX.md](../tickets/INDEX.md) interleaves the phases for this reason.
- This phase introduces the first migrations after `001_initial.sql`. `001_initial.sql` is frozen; every schema change is a new numbered file. Migration numbers are allocated in ticket order: 002 (GRF-212), 003 (GRF-213), 004 (GRF-214), 007 (GRF-215). If tickets land out of order, renumber to match the actual apply order and record it here.
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

