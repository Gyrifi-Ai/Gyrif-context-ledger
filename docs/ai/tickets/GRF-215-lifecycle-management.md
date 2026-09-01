# GRF-215 — Ledger and Change lifecycle management

| Field | Value |
|---|---|
| Type | Story |
| Phase | 2 — Governance API completeness |
| Epic | Governance API |
| Priority | Medium |
| Size | M |
| Depends on | GRF-212 |
| Blocks | — |

## Summary

Give operators a supported way to retire a Ledger and to discard a Change that was ingested by mistake. Both entities are currently create-only, so every error is permanent.

## Context

Verified against `runtime/internal/interfaces/http/server.go` `routes()` — there is no `DELETE` route anywhere in the API. `Repository` has no delete method for any entity.

Consequences:

- A Ledger created with a typo in its name is permanent and clutters the switcher forever.
- A Change ingested from a misconfigured pipeline — wrong unit id, wrong collection, test data against production — sits in the inbox permanently. It cannot be released (nobody wants it) and cannot be removed. Once GRF-214 lands it will be paginated, but it will still be there.
- The only escape is to propose the unwanted Changes and release them, which writes garbage to Qdrant, or to leave them in the inbox forever, which trains operators to ignore the inbox. Both outcomes are bad, and the second is worse.

## The governing principle

**History is immutable. Intent is not.**

A Change that has been released is part of the audit trail and must never be removable. A Change that has never been released expresses an intention that turned out to be wrong, and withdrawing an intention is a legitimate governance act — one that is itself recorded.

So this ticket adds **withdrawal**, not deletion. Nothing is erased; a terminal state is recorded.

## Scope

### In scope

- `POST /api/v1/ledgers/{ledgerID}/changes/{changeID}/withdraw`
- `POST /api/v1/ledgers/{ledgerID}/archive` and `.../unarchive`
- Filtering archived Ledgers out of the default list.
- Studio actions for both.

### Out of scope

- Hard deletion of any entity. If a regulator or a GDPR request ever requires erasure, that is a distinct, carefully-designed feature — not this ticket. Object reclamation for withdrawn Changes is handled by GRF-222's reachability rules.
- Editing a Change. Changes are immutable by design; withdraw and re-ingest.

## Acceptance criteria

**Change withdrawal**

- [x] New `ChangeStatus` member `WITHDRAWN`.
- [x] `ledger.CanWithdrawChange(c ledger.Change) error` added to `invariants.go` as the pure rule.
- [x] Withdrawal is permitted only when the Change's status is `ACCEPTED`, `READY`, or `INVALID`.
- [x] Withdrawing a `RELEASED` Change ⇒ `409 CONFLICT` "A released Change is part of the audit trail and cannot be withdrawn."
- [x] Withdrawing a Change claimed by a Proposal ⇒ `409 CONFLICT` "This Change belongs to Proposal {id}. Cancel the Proposal first." — pointing at GRF-212.
- [x] Withdrawing an already-`WITHDRAWN` Change is idempotent and returns `204`.
- [x] A `WITHDRAWN` Change can never be selected into a Proposal; `CreateProposal` rejects it with `409` naming the status.
- [x] The Change row is **retained**, with its idempotency key intact — so a repeat of the same erroneous ingestion still deduplicates rather than creating a second copy.
- [x] Withdrawal records who and when: new columns `withdrawn_at TEXT`, `withdrawn_reason TEXT`.
- [x] The request body requires a non-empty `reason`; an empty or missing reason ⇒ `400 INVALID_ARGUMENT`.

**Ledger archival**

- [x] New column `archived_at TEXT` on `ledgers`.
- [x] `POST .../archive` sets it; `POST .../unarchive` clears it. Both return `204` and are idempotent.
- [x] Archiving is rejected with `409 CONFLICT` when the Ledger has a Proposal in `DRAFT` or a Release Intent that is not `FINALIZED` or `ABANDONED` — you may not archive a Ledger with work in flight.
- [x] `GET /api/v1/ledgers` excludes archived Ledgers by default and includes them with `?includeArchived=true`.
- [x] An archived Ledger rejects new Changes and new Proposals with `409 CONFLICT` "This Ledger is archived."
- [x] An archived Ledger remains fully **readable** — its Changes, Proposals, and Releases are all still retrievable. Archival hides it from the working set; it does not hide history.
- [x] Rollback against an archived Ledger is rejected; unarchive first.

**Schema and general**

- [x] Migration `007_lifecycle.sql` (renumber to match actual apply order; record the number in the phase log). `001_initial.sql` is not modified.
- [x] `WITHDRAWN` added to the frontend status union and to the `design-system.md` §2.2 tone map as `neutral`. The exhaustive TypeScript mapping must fail to compile until it is added.
- [x] Studio: a `Withdraw` action in the Change detail drawer behind a `ConfirmDialog` with a required reason field; withdrawn Changes are hidden from the inbox by default and reachable via the status filter.
- [x] Studio: `Archive` in the Ledgers page row actions behind a `ConfirmDialog`; an `Archived` toggle reveals archived Ledgers; the topbar switcher excludes them.
- [x] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- Status guards must be inside the transaction and re-read on the write connection — no check-then-act across statements.
- Do **not** delete the object store entry for a withdrawn Change here. GRF-222 owns reclamation, and its reachability rules must be updated to treat a withdrawn, unreleased Change's value as unreachable after the grace period. Note this dependency in both phase log entries.
- Withdrawal is a state transition, not a deletion — resist any suggestion to `DELETE FROM changes`. The retained idempotency key is the specific reason.
- If GRF-210 has landed, publish `change.withdrawn` and `ledger.archived`.
- Archival is deliberately reversible. Nothing about it is destructive, which is why it does not need the ceremony that a real deletion would.

## Test plan

- `runtime/internal/ledger/invariants_test.go` — `CanWithdrawChange` for every status.
- `runtime/tests/change_withdrawal_test.go`:
  - withdraw a `READY` Change ⇒ 204; it disappears from the default inbox and cannot be proposed,
  - withdraw a claimed Change ⇒ 409 naming the proposal,
  - cancel the proposal (GRF-212) then withdraw ⇒ 204,
  - withdraw a `RELEASED` Change ⇒ 409,
  - withdraw twice ⇒ idempotent,
  - re-ingest with the same idempotency key after withdrawal ⇒ returns the withdrawn Change, does not create a duplicate,
  - missing reason ⇒ 400.
- `runtime/tests/ledger_archive_test.go`:
  - archive with a draft proposal ⇒ 409,
  - archive a quiet ledger ⇒ 204; excluded from the default list; included with the flag,
  - archived ledger rejects changes and proposals but serves all reads,
  - unarchive restores full function.

## Docs to update

- `docs/ai/product.md` §2 (Change and Ledger statuses), §3, §5 (withdrawal and archival as governed transitions), §7.
- `docs/ai/tech-spec.md` §3 (endpoints), §4 (`ChangeStatus` gains `WITHDRAWN`), §7/§8 (schema, migration).
- `docs/ai/design-system.md` §2.2 and §5.1/§5.2.
- `docs/ai/tickets/GRF-222-retention-backup.md` — reachability must account for withdrawn Changes.
- `docs/ai/phases/phase-2.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
