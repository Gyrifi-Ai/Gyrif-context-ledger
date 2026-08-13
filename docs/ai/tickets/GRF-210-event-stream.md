# GRF-210 — Real domain event stream

| Field | Value |
|---|---|
| Type | Story |
| Phase | 2 — Governance API completeness |
| Epic | Governance API |
| Priority | High |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Turn `/events/v1` from a keepalive stub into a real server-sent event stream that lets Studio react to Changes, Proposals, and Releases without polling.

## Context

`runtime/internal/interfaces/http/server.go`, `func (server *Server) events`:

```go
writer.Header().Set("Content-Type", "text/event-stream")
writer.Header().Set("Cache-Control", "no-cache")
fmt.Fprint(writer, "event: ledger\ndata: {\"status\":\"connected\"}\n\n")
flusher.Flush()
ticker := time.NewTicker(20 * time.Second)
// ... only ": keepalive" from here
```

No domain event is ever emitted. `studio/src/api/events.ts` exports `subscribeToEvents` which nothing imports. Every Studio screen therefore shows stale data until the user reloads.

## Scope

### In scope

- An in-process publish/subscribe broker owned by the Engine.
- Event emission at every state transition that a user can observe.
- SSE fan-out with per-subscriber buffering, slow-consumer handling, and ledger filtering.
- `Last-Event-ID` support is **out of scope** — see below.

### Out of scope

- Event persistence, replay, or `Last-Event-ID` resume. State is derivable from the REST API; the stream is a hint to refetch, not a source of truth.
- A message queue, an event bus library, or any new dependency.
- Cross-process fan-out. Gyrifi is explicitly single-replica.

## Design

Add `runtime/internal/engine/events.go`:

```go
type EventKind string

const (
    EventChangeAccepted   EventKind = "change.accepted"
    EventProposalCreated  EventKind = "proposal.created"
    EventProposalEvaluated EventKind = "proposal.evaluated"
    EventProposalApproved EventKind = "proposal.approved"
    EventReleaseStarted   EventKind = "release.started"
    EventReleaseCompleted EventKind = "release.completed"
    EventReleaseFailed    EventKind = "release.failed"
    EventIntentRecoveryRequired EventKind = "intent.recovery_required"
)

type Event struct {
    Kind     EventKind `json:"kind"`
    LedgerID string    `json:"ledgerId"`
    SubjectID string   `json:"subjectId"`  // change/proposal/release/intent id
    At       time.Time `json:"at"`
}

type Broker struct { /* mu, map[int]chan Event, nextID */ }
func (b *Broker) Subscribe(buffer int) (<-chan Event, func())
func (b *Broker) Publish(Event)
```

The `Engine` owns a `*Broker`. `Publish` is **non-blocking**: if a subscriber's buffered channel is full, drop the event for that subscriber and increment a dropped counter. A dropped event is acceptable because the client's response to any event is "refetch".

The HTTP handler subscribes, optionally filters by a `?ledgerId=` query parameter, and writes:

```text
event: change.accepted
data: {"kind":"change.accepted","ledgerId":"ldg_…","subjectId":"chg_…","at":"2026-08-12T10:00:00Z"}
```

## Acceptance criteria

- [ ] `engine.Broker` exists with `Subscribe(buffer int) (<-chan Event, func())` and a non-blocking `Publish`.
- [ ] `Publish` never blocks and never panics on a closed channel; the unsubscribe function is idempotent.
- [ ] Events are published from: `CreateChange` (new only, not idempotent replays), `CreateProposal`, `EvaluateProposal`, `ApproveProposal`, `ReleaseProposal` (started / completed / failed), `RecoverReleases` (recovery-required).
- [ ] Publishing happens **after** the durable write commits, never before.
- [ ] Publishing failures cannot fail a governance operation. No `Publish` call returns an error into the request path.
- [ ] `GET /events/v1` supports an optional `ledgerId` query parameter and only forwards matching events when it is present.
- [ ] The initial `event: ledger` / `{"status":"connected"}` frame is preserved for backwards compatibility.
- [ ] `: keepalive` continues every 20 s.
- [ ] The handler flushes after every event and exits cleanly on `request.Context().Done()`, always calling unsubscribe.
- [ ] A slow or disconnected client cannot block the Engine — verified by a test that subscribes with `buffer: 1`, publishes 100 events without reading, and asserts `Publish` returns promptly.
- [ ] `studio/src/api/events.ts` is updated to expose `subscribeToLedgerEvents(ledgerId, handler)` typed against the event payload, and is wired into the pages via the GRF-204 `useLedgerEvents` hook.
- [ ] Studio refetches the affected query on each event; it does **not** apply the event payload as state.
- [ ] `go test ./...`, `pnpm test` pass.

## Implementation notes

- Keep the broker in `engine/` — it is application-level, not a protocol concern, and both HTTP and any future interface should be able to use it.
- Use a `map[uint64]chan Event` plus a `sync.RWMutex`. Do not reach for a channel-of-channels design.
- `Publish` pattern:
  ```go
  select {
  case ch <- event:
  default:
      b.dropped.Add(1)
  }
  ```
- Log dropped counts at `debug` only.
- In `ReleaseProposal`, publish `release.started` after the intent is persisted, and `release.failed` in both the apply and verify failure branches before returning.
- Studio must treat the stream as advisory: an event means "something changed, refetch", never "here is the new state". This keeps the server authoritative for all governance rules.

## Test plan

- `runtime/internal/engine/events_test.go` — subscribe/publish/unsubscribe; non-blocking under a full buffer; idempotent unsubscribe.
- `runtime/internal/interfaces/http/events_test.go` — `httptest` request, assert the connected frame, a forwarded event, ledger filtering, and clean shutdown on context cancel.
- Extend `runtime/tests/change_flow_test.go` to assert the expected event sequence for a full flow.
- `studio/src/api/events.test.ts` — event parsing and handler dispatch with a stubbed `EventSource`.

## Docs to update

- `docs/ai/tech-spec.md` §3 (`/events/v1` behaviour) and §11 (frontend contract) — replace the "stub" description with the real contract and the event table.
- `docs/ai/product.md` §7 — remove the event-stream gap row.
- `docs/ai/repo-structure.md` — add `internal/engine/events.go`.
- `docs/ai/phases/phase-2.md` — completion entry with the final event kind list.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
