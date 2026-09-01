# GRF-221 — Asynchronous Change preparation and base fingerprint

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Correctness |
| Priority | High |
| Size | L |
| Depends on | — |
| Blocks | — |

## Summary

Populate `Change.baseFingerprint` by reading the target's current state, and move that read out of the ingestion request path into a background preparation worker.

## Context

`ChangeStatus` defines `ACCEPTED`, `READY`, `INVALID`, and `RELEASED`, which describes a preparation pipeline. In practice `CreateChange` inserts the Change directly as `READY` and sets `baseFingerprint` to the empty string. The `ACCEPTED` status is written but immediately superseded, and `INVALID` is never written at all.

Consequences:

- The system cannot tell whether the desired state differs from what the target already holds. A no-op Change is indistinguishable from a real one.
- Release-time drift detection relies solely on `expectedFingerprint` computed at plan time, so a target mutation between ingestion and release is caught late, at apply, rather than early.
- The Changes inbox cannot show "no change needed", so operators review noise.

The obvious fix — read the target inside `CreateChange` — is wrong: it puts a network call to Qdrant inside the ingestion hot path and inside a SQLite transaction.

## Scope

### In scope

- A preparation worker that transitions `ACCEPTED → READY` or `ACCEPTED → INVALID`.
- Populating `baseFingerprint` from the target's observed value.
- A `NOOP` outcome for Changes whose desired state already matches the target.
- Reclaimable ownership so a crash mid-preparation does not strand Changes.
- Studio surfacing of `ACCEPTED` and `INVALID`.

### Out of scope

- Multi-process workers. Gyrifi is single-replica; a single in-process worker with reclaim is sufficient.
- Retry backoff tuning beyond a simple bounded exponential schedule.

## Design

`CreateChange` inserts with status `ACCEPTED` and returns `202` as it does today. The response contract does not change.

A worker goroutine, started by `bootstrap` after the recovery sweep, loops:

1. Claim a batch of `ACCEPTED` Changes whose `prepare_after` has elapsed, setting `prepare_owner` and `prepare_claimed_at` in one transaction.
2. **Outside any transaction**, read the current values for those units from the target adapter.
3. In a new transaction, write `base_fingerprint` and the resulting status.

Outcomes:

| Observation | Result |
|---|---|
| Unit absent, action `PUT` | `READY`, `base_fingerprint = ""` |
| Unit present, fingerprint ≠ desired | `READY`, `base_fingerprint = observed` |
| Unit present, fingerprint = desired, action `PUT` | `READY` with `noop = true` |
| Unit absent, action `DELETE` | `READY` with `noop = true` |
| Target unreachable / 5xx | stay `ACCEPTED`, increment attempts, set `prepare_after` |
| Desired value rejected by the adapter (dimension mismatch, unsupported type) | `INVALID` with a stored reason |

**Retryable and permanent must not be conflated.** A target outage is not an invalid Change. Only an adapter-reported semantic rejection produces `INVALID`.

`noop` is surfaced but does **not** block proposing — an operator may legitimately want a no-op Change recorded in history.

## Acceptance criteria

- [x] Migration `006_change_preparation.sql` adds to `changes`: `prepare_owner TEXT`, `prepare_claimed_at TEXT`, `prepare_attempts INTEGER NOT NULL DEFAULT 0`, `prepare_after TEXT`, `invalid_reason TEXT`, `noop INTEGER NOT NULL DEFAULT 0`. Existing rows are backfilled to `READY` and untouched otherwise.
- [x] `CreateChange` inserts `ACCEPTED` and still returns `202` with the same body shape.
- [x] Idempotent replays still return the existing Change without re-queuing preparation.
- [x] The worker lives in `runtime/internal/engine/preparation.go` and is started and stopped by `bootstrap` with the process context.
- [x] **No target I/O occurs inside a SQLite transaction.** A code review checklist item and, where feasible, a test using a target fake that asserts the DB is not holding a write transaction during its call.
- [x] Claims are reclaimed when `prepare_claimed_at` is older than a bounded lease (default 2 minutes), so a crash cannot strand a Change permanently.
- [x] Retryable failures use bounded exponential backoff (e.g. 1s, 2s, 4s … capped at 5 minutes) and never transition to `INVALID`.
- [x] After a configurable attempt ceiling the Change stays `ACCEPTED` and is surfaced as stalled; it is **not** marked `INVALID`.
- [x] `INVALID` is written only for adapter-reported semantic rejections, always with a non-empty `invalid_reason`.
- [x] `base_fingerprint` is computed with the same `ledger.Fingerprint` function used elsewhere — no parallel implementation.
- [x] Proposal creation rejects Changes that are not `READY` with `409 CONFLICT` naming the offending status.
- [x] The worker is idle-cheap: no busy loop, no per-second polling of an empty queue. Use a ticker with a modest interval plus a wake channel signalled by `CreateChange`.
- [x] Shutting the process down mid-preparation leaves no Change in an unrecoverable state, verified by a test.
- [x] New config: `GYRIFI_PREPARE_BATCH_SIZE` (default 25), `GYRIFI_PREPARE_LEASE` (default `2m`), validated like the existing config values.
- [x] Studio Changes inbox renders `ACCEPTED` (with a "Preparing" label and a `Skeleton` fingerprint), `INVALID` (with the reason), and a `No change needed` marker for `noop` rows. The status filter includes all four.
- [x] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- The worker owns a target adapter obtained the same way the release path does. If no target is configured, preparation should mark Changes `READY` with an empty base fingerprint and log once — the product must remain usable without a target.
- Batch the target reads per collection where the adapter supports it; do not issue one HTTP request per unit if a batch read exists.
- `prepare_owner` can be a per-process UUID generated at startup. It exists for reclaim diagnostics, not for coordination.
- Keep `ledger/` pure — the outcome decision table belongs in `engine/` or as a pure function in `ledger/invariants.go` taking observed and desired fingerprints. Prefer the latter and unit-test it in isolation.
- If GRF-210 has landed, publish `change.ready` and `change.invalid`.

## Test plan

- `runtime/internal/ledger/preparation_outcome_test.go` — the decision table above, exhaustively, as a pure function test.
- `runtime/tests/change_preparation_test.go` with a controllable target fake:
  - absent unit ⇒ `READY`, empty base fingerprint,
  - differing unit ⇒ `READY` with the observed fingerprint,
  - identical unit ⇒ `READY` + `noop`,
  - `DELETE` on an absent unit ⇒ `READY` + `noop`,
  - target returning 503 ⇒ stays `ACCEPTED`, attempts increment, backoff respected, and succeeds once the fake recovers,
  - adapter semantic rejection ⇒ `INVALID` with a reason,
  - proposing an `ACCEPTED` or `INVALID` Change ⇒ 409,
  - a claim with an expired lease is reclaimed by the next tick,
  - no target configured ⇒ Changes still reach `READY`.
- Concurrency test: 200 Changes ingested at once are all prepared exactly once.

## Docs to update

- `docs/ai/product.md` §2 (Change lifecycle now real), §3 (workflow step 2), §5 (new invariant: only `READY` Changes may be proposed), §7 (remove the base-fingerprint gap).
- `docs/ai/tech-spec.md` §2 (new config), §6 (worker), §7/§8 (schema, migration 006).
- `docs/ai/repo-structure.md` — `internal/engine/preparation.go`.
- `docs/ai/design-system.md` §5.2 — `ACCEPTED`/`INVALID`/`noop` presentation.
- `docs/ai/phases/phase-3.md` — completion entry with the outcome table as implemented.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
