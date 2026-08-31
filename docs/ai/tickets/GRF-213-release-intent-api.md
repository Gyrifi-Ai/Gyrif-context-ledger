# GRF-213 — Release intent inspection and recovery API

| Field | Value |
|---|---|
| Type | Story |
| Phase | 2 — Governance API completeness |
| Epic | Governance API |
| Priority | Highest |
| Size | L |
| Depends on | — |
| Blocks | GRF-208 |

## Summary

Expose Release Intents for reading and give operators an explicit way to resolve a `RECOVERY_REQUIRED` intent. Today a failed release leaves an invisible, unresolvable row and a ledger that cannot release again.

## Context

`ReleaseIntent` is the durable record of an in-flight apply, holding the compiled `Plan` with expected fingerprints and retained before-images. Its lifecycle is `READY → APPLYING → VERIFYING → FINALIZED`, or `RECOVERY_REQUIRED` on failure.

Current state:

- `Repository` has `InsertReleaseIntent`, `UpdateReleaseIntentStatus`, and `ListReleaseIntentsByStatus`. There is **no** way to load a single intent or list a ledger's intents.
- `bootstrap` calls a recovery sweep on startup, but nothing surfaces intents that stay `RECOVERY_REQUIRED`.
- No HTTP endpoint mentions intents at all.
- The operator sees a `503` from the release call and then nothing, forever. The retained before-images — the entire basis for safe rollback — are unreadable.
- Studio's Releases page cannot show the plan because the plan lives only on the intent.

## Scope

### In scope

- `GET /api/v1/ledgers/{ledgerID}/release-intents` (optional `?status=`)
- `GET /api/v1/ledgers/{ledgerID}/release-intents/{intentID}`
- `POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/retry`
- `POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/resolve`
- Repository and Engine methods.
- Recovery classification that is explicit rather than implied.

### Out of scope

- Automatic background retry loops. Recovery is operator-initiated after the startup sweep; silent retries against a target that may have partially applied are not acceptable.
- Changing the release algorithm itself.

## Recovery semantics — read before implementing

An intent reaches `RECOVERY_REQUIRED` when the runtime cannot prove the target's state. Two distinct situations hide behind that one status:

1. **Verification never completed** — the apply may or may not have landed. The correct action is to re-verify against the target.
2. **Verification completed and disagreed** — the target holds values that are neither the expected pre-state nor the desired state. Something outside Gyrifi wrote to the collection. No automated action is safe.

The API must distinguish these, because the safe response differs.

- `retry` re-runs **verification only**, never re-applies. If verification now passes, the intent finalizes and the release completes normally. If it fails, the intent stays `RECOVERY_REQUIRED` and the response reports the mismatched units.
- `resolve` is an explicit operator override that marks the intent abandoned. It requires a body with `{ "resolution": "ABANDONED", "note": "…" }` and a non-empty note. It does **not** finalize the release and does **not** advance `HEAD`.

## Acceptance criteria

**Read**

- [x] `GET .../release-intents` returns `{ "items": [ReleaseIntent] }` newest-first, filtered by `status` when the query parameter is present and valid. An unknown status ⇒ `400 INVALID_ARGUMENT`.
- [x] `GET .../release-intents/{intentID}` returns the intent including the full `Plan` with per-operation `unit`, `action`, `expectedFingerprint`, `desiredFingerprint`, and `hasBeforeImage`.
- [x] `hasBeforeImage` is derived by checking that the retained before-image object is present in the object store, not by assuming it.
- [x] An intent belonging to another ledger ⇒ `404 NOT_FOUND`.
- [x] `Repository` gains `LoadReleaseIntent(ctx, id string) (ledger.ReleaseIntent, error)` and `ListReleaseIntentsForLedger(ctx, ledgerID string, status *ledger.ReleaseIntentStatus) ([]ledger.ReleaseIntent, error)`.

**Retry**

- [x] `POST .../retry` is valid only for `RECOVERY_REQUIRED` and `VERIFYING`; any other status ⇒ `409 CONFLICT` naming the current status.
- [x] Retry re-reads the target and compares against the plan's desired fingerprints. It **never** issues a write to the target.
- [x] On success the intent moves to `FINALIZED`, the Release row is completed, `HEAD` advances, the proposal becomes `RELEASED`, and the affected Changes become `RELEASED` — using the same code path as the happy-path finalisation, not a copy.
- [x] On failure the response is `200` with `{ "resolved": false, "mismatches": [{ "unit", "expected", "observed" }] }` and the intent stays `RECOVERY_REQUIRED`.
- [x] Target unreachable during retry ⇒ `503 UNAVAILABLE`, intent unchanged.
- [x] Retry is safe to call repeatedly.

**Resolve**

- [x] `POST .../resolve` requires `{ "resolution": "ABANDONED", "note": "<non-empty>" }`. A missing or empty note ⇒ `400 INVALID_ARGUMENT`.
- [x] Any resolution value other than `ABANDONED` ⇒ `400 INVALID_ARGUMENT`.
- [x] Resolving sets the intent status to `ABANDONED` (new status), records the note and timestamp, does not advance `HEAD`, and leaves the Proposal in its pre-release status so it can be re-released after the operator repairs the target.
- [x] Resolve is rejected with `409` unless the current status is `RECOVERY_REQUIRED`.
- [x] A ledger with an unresolved `RECOVERY_REQUIRED` intent rejects new releases with `409 CONFLICT` "A release intent requires recovery before further releases." — add this guard to `ReleaseProposal` if it is not already enforced.

**Schema and general**

- [x] Migration `002_release_intent_resolution.sql` adds `resolution TEXT`, `resolution_note TEXT`, `resolved_at TEXT` to `release_intents`. `ABANDONED` added to the status set in code. The planned `003` number was renumbered because GRF-213 landed before GRF-212.
- [x] `go test ./...` passes; no existing release test regresses.

## Implementation notes

- Factor the existing finalisation tail of `ReleaseProposal` into `func (engine *Engine) finalizeIntent(ctx, intent) error` and call it from both the happy path and `retry`. Two implementations of finalisation will diverge and corrupt history.
- The startup sweep should keep doing what it does; this ticket only adds the operator-facing surface on top of the same primitives.
- Verification comparison must reuse the target adapter's existing fingerprint comparison, including the 1e-6 cosine tolerance. Do not re-implement float comparison.
- Return the intent's `Plan` as-is; it is already a serialisable domain type.
- If GRF-210 has landed, publish `intent.recovery_required` on entry and a resolution event on exit.
- Guard against a ledger having multiple concurrent unresolved intents — the release guard above makes this impossible going forward, but the list endpoint must handle historical rows.

## Test plan

- `runtime/tests/release_recovery_test.go` using a target fake that can be switched to fail mid-apply and mid-verify:
  - failure during verify ⇒ intent `RECOVERY_REQUIRED`, visible via both endpoints,
  - `retry` when the target is actually correct ⇒ finalizes; `HEAD` advances exactly once,
  - `retry` when the target holds foreign values ⇒ `resolved: false` with the mismatch list,
  - `retry` on a `FINALIZED` intent ⇒ 409,
  - `resolve` without a note ⇒ 400; with a note ⇒ `ABANDONED`, `HEAD` unchanged,
  - a second `ReleaseProposal` while an intent is `RECOVERY_REQUIRED` ⇒ 409,
  - after `resolve`, releasing the same proposal again succeeds.
- Assert `hasBeforeImage` is false when the object is absent from the store.
- Cross-ledger id ⇒ 404.

## Docs to update

- `docs/ai/tech-spec.md` §3 (four endpoints), §4 (`ReleaseIntentStatus` gains `ABANDONED`), §7 (Repository), §8 (migration 003).
- `docs/ai/product.md` §4 and §5 — document the recovery workflow and the new release guard as an invariant.
- `docs/ai/phases/phase-2.md` — completion entry; record the retry-never-writes decision prominently.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
