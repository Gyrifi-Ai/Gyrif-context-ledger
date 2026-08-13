# GRF-212 — Proposal cancellation and claim release

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

Allow a Proposal to be cancelled, releasing its claims so the Changes become proposable again. Today a mistaken Proposal permanently strands every Change it contains.

## Context

`runtime/migrations/001_initial.sql`:

```sql
CREATE TABLE proposal_changes (
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    change_id TEXT NOT NULL REFERENCES changes(id),
    ordinal INTEGER NOT NULL,
    PRIMARY KEY (proposal_id, change_id),
    UNIQUE (change_id)                 -- one Proposal per Change, forever
);
```

`UNIQUE (change_id)` enforces [invariant 4](../product.md). There is no code path that deletes a claim. Consequences:

- A Proposal created with the wrong Changes or the wrong order can never be corrected.
- Those Changes can never be released, because `CreateProposal` will always hit the unique constraint.
- The only workaround is re-submitting the same desired state under a new idempotency key, which pollutes the audit trail.

`ProposalStatus` already defines `BLOCKED`, which is currently never written.

## Scope

### In scope

- `POST /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/cancel`
- A `CANCELLED` proposal status (new) and the transactional claim release.
- A guard preventing cancellation once release has started.
- Studio wiring: enable the `Cancel` action stubbed out in GRF-207.

### Out of scope

- Editing proposal membership in place. Cancel-and-recreate is the supported path; in-place editing would silently invalidate the hash, checks, and approvals, and is a larger design question.
- Deleting proposals. History is retained; cancellation is a state, not a deletion.

## Design decisions

**Add `CANCELLED`, do not reuse `BLOCKED`.** `BLOCKED` reads as "cannot proceed right now"; `CANCELLED` is terminal and intentional. Add:

```go
ProposalCancelled ProposalStatus = "CANCELLED"
```

**Keep the claim rows, add a release marker?** No. Deleting the `proposal_changes` rows is correct: the claim's purpose is to enforce exclusivity, and a cancelled proposal has no claim. The audit trail is preserved by the proposal row itself plus its recorded `changeIds`.

To keep the membership readable after cancellation, add a `change_ids` snapshot column to `proposals`:

```sql
-- runtime/migrations/002_proposal_cancellation.sql
ALTER TABLE proposals ADD COLUMN change_ids TEXT NOT NULL DEFAULT '[]';
```

Backfill it from `proposal_changes` in the same migration for existing rows.

## Acceptance criteria

- [ ] New migration `runtime/migrations/002_proposal_cancellation.sql`. `001_initial.sql` is **not** modified.
- [ ] The migration backfills `proposals.change_ids` from `proposal_changes` ordered by `ordinal` for every existing proposal.
- [ ] `InsertProposal` writes the `change_ids` JSON snapshot alongside the claim rows, in the same transaction.
- [ ] `LoadProposal` and `ListProposals` read `changeIds` from the snapshot column, so cancelled proposals still report their membership.
- [ ] `ledger.ProposalCancelled` added; `ledger.CanCancelProposal(p ledger.Proposal) error` added to `invariants.go` as the pure rule.
- [ ] `Repository.CancelProposal(ctx, ledgerID, proposalID string) error` performs, in one transaction: verify status is `DRAFT`, verify no `release_intents` row exists for the proposal, delete the `proposal_changes` rows, set the proposal status to `CANCELLED`.
- [ ] `Engine.CancelProposal(ctx, ledgerID, proposalID string) error` wraps it with `CodeNotFound` / `CodeConflict`.
- [ ] Cancelling a `RELEASED` proposal ⇒ `409 CONFLICT` "A released Proposal cannot be cancelled."
- [ ] Cancelling a proposal with any existing `release_intents` row ⇒ `409 CONFLICT` "Release has already started for this Proposal."
- [ ] Cancelling an already-`CANCELLED` proposal is idempotent and returns `204`.
- [ ] After cancellation the affected Changes have status `READY` and can be selected into a new Proposal.
- [ ] `POST .../cancel` returns `204 No Content`.
- [ ] Checks and approvals for the cancelled proposal are **retained** (audit), and are never considered current for any other proposal because they carry the old hash.
- [ ] `features/shared/status.ts` maps `CANCELLED` to the `neutral` tone; the exhaustive switch still compiles.
- [ ] Studio: the `Cancel` action in the proposal detail is enabled, behind a `ConfirmDialog` explaining that the Changes return to the inbox.
- [ ] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- The status guard must be inside the transaction, re-read with the write connection — do not check-then-act across statements.
- Deleting `proposal_changes` rows is what frees the `UNIQUE (change_id)` constraint. Verify with a test that immediately re-proposes the same Change IDs.
- Do **not** relax or drop `UNIQUE (change_id)`. That constraint is [invariant 4](../product.md) and is load-bearing.
- If GRF-210 has landed, publish a `proposal.cancelled` event after commit.
- SQLite `ALTER TABLE ... ADD COLUMN` with a `NOT NULL DEFAULT` is safe and does not rewrite the table.

## Test plan

- `runtime/tests/proposal_cancellation_test.go`:
  - cancel a `DRAFT` proposal ⇒ 204; its Changes are `READY`; a new Proposal with the same Change IDs succeeds,
  - cancel twice ⇒ idempotent 204,
  - cancel a `RELEASED` proposal ⇒ 409,
  - cancel a proposal with a persisted intent ⇒ 409,
  - checks and approvals survive cancellation and are not current for the replacement proposal (different hash).
- Migration test: seed a DB at migration 001 with proposals and claims, apply 002, assert `change_ids` is backfilled in ordinal order.

## Docs to update

- `docs/ai/tech-spec.md` §3 (endpoint), §4 (`ProposalStatus` gains `CANCELLED`), §8 (schema — new column and migration), §7 (Repository interface).
- `docs/ai/product.md` §2 (Proposal statuses), §7 (remove the cancellation gap row).
- `docs/ai/design-system.md` §2.2 — add the `CANCELLED` status mapping.
- `docs/ai/phases/phase-2.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
