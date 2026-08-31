# GRF-211 — Proposal detail and evidence read API

| Field | Value |
|---|---|
| Type | Story |
| Phase | 2 — Governance API completeness |
| Epic | Governance API |
| Priority | Highest |
| Size | M |
| Depends on | — |
| Blocks | GRF-207 |

## Summary

Add read endpoints for a single Proposal, its Changes, its evaluation evidence, and its approvals. Without these, the review workspace (GRF-207) cannot show the governance state that the server actually holds.

## Context

Evidence and approvals are written but never readable:

- `checks` and `approvals` rows are inserted by `SaveCheckResult` and `SaveApproval`.
- The only reads are the booleans `HasPassingCheck(proposalID, proposalHash)` and `HasApproval(proposalID, proposalHash)`.
- `POST .../evaluation` returns the result **once**; reload the page and it is gone.
- `findings`, `previewFidelity`, the model identity, and the bound hash are all invisible after the response is consumed.
- There is no `GET` for a single Proposal at all — Studio filters the list client-side.

## Scope

### In scope

- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}`
- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/checks`
- `GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals`
- Corresponding `Repository` and `Engine` methods.
- TypeScript contract types and `api` client methods.

### Out of scope

- Writing new evidence kinds.
- Any schema change — every field needed already exists in `checks` and `approvals`.
- Redaction policy for evidence blobs (see notes).

## API contract

### `GET .../proposals/{proposalID}`

`200`:

```json
{
  "proposal": { "...": "ledger.Proposal" },
  "changes": [ { "...": "ledger.Change" } ],
  "currentHeadReleaseId": "rel_…",
  "gates": {
    "hasCurrentPassingCheck": true,
    "hasCurrentApproval": false,
    "baseMatchesHead": true,
    "releasable": false,
    "reason": "A current approval is required."
  }
}
```

`gates` is computed by the Engine using the same predicates the release path uses. **Studio renders `reason` verbatim and never re-derives it.**

### `GET .../proposals/{proposalID}/checks`

`200`: `{ "items": [ { "id", "proposalHash", "kind", "passed", "summary", "evidence", "createdAt", "current": true } ] }`

Newest first. `current` is `proposalHash === proposal.hash`. `evidence` is the stored blob decoded as JSON when it parses, otherwise omitted with `"evidenceUnavailable": true`.

### `GET .../proposals/{proposalID}/approvals`

`200`: `{ "items": [ { "id", "proposalHash", "actor", "createdAt", "current": true } ] }`

## Acceptance criteria

- [ ] Three `GET` endpoints registered in `server.routes()` with the exact paths above.
- [ ] `Repository` gains `ListCheckResults(ctx, proposalID string) ([]ledger.CheckResult, error)` and `ListApprovals(ctx, proposalID string) ([]ledger.Approval, error)`, both newest-first.
- [ ] `Engine` gains `LoadProposalDetail(ctx, ledgerID, proposalID string) (ProposalDetail, error)`, `ListCheckResults`, and `ListApprovals`.
- [ ] `ProposalDetail.Gates` is computed with the **same** predicates as `ReleaseProposal`: `HasPassingCheck`, `HasApproval`, and `head.ReleaseID == proposal.BaseReleaseID`. Extract them into one shared unexported helper so the two paths cannot drift.
- [ ] `reason` messages are identical to the ones `ReleaseProposal` returns for the same condition.
- [ ] A proposal id belonging to a different ledger returns `404 NOT_FOUND`, not another ledger's data.
- [ ] `CheckResult.Evidence` is decoded to JSON when valid; malformed blobs yield `evidenceUnavailable: true` instead of a 500.
- [ ] `Change.idempotencyKey` and `Change.requestFingerprint` remain unserialised (they are `json:"-"`) — verify no new struct re-exposes them.
- [ ] TypeScript types added to `studio/src/api/types.ts`: `CheckResult`, `Approval`, `ProposalGates`, `ProposalDetail`.
- [ ] `api.proposal(ledgerId, proposalId)`, `api.proposalChecks(...)`, `api.proposalApprovals(...)` added to `studio/src/api/client.ts`.
- [ ] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- Add the SQL in `runtime/internal/repository/sqlite.go` using the existing `checks_current` index for ordering.
- Extract the gate logic:
  ```go
  func (engine *Engine) evaluateGates(ctx context.Context, proposal ledger.Proposal) (Gates, error)
  ```
  and call it from both `LoadProposalDetail` and `ReleaseProposal`. `ReleaseProposal` keeps its own error wrapping but must use the same booleans and message strings.
- Do **not** add a `status` transition here. Proposal statuses `REVIEWED`/`APPROVED` remain unused; changing them is a separate decision.
- **Evidence redaction:** the deterministic evidence blob is the full effective proposed state, which contains every desired value in the proposal. That is the same data an admitted caller can read via `GET /changes`. ADR 0002 superseded GRF-220's application-auth design; the deployment boundary protects both surfaces.
- Keep response assembly in the handler thin; the Engine returns the struct.

## Test plan

- `runtime/internal/repository` tests: list checks/approvals ordering and ledger scoping.
- `runtime/tests/proposal_detail_test.go`:
  - detail before evaluation ⇒ `releasable: false`, reason mentions evaluation,
  - after evaluation ⇒ reason mentions approval,
  - after approval ⇒ `releasable: true`,
  - after a hash-changing scenario ⇒ `hasCurrentPassingCheck: false`,
  - cross-ledger id ⇒ 404.
- Assert the gate `reason` string equals the `ReleaseProposal` error message for the same state — this is the anti-drift test.
- `studio/src/api/client.test.ts` — new endpoint paths.

## Docs to update

- `docs/ai/tech-spec.md` §3 (endpoint table), §4 (new wire types), §6 (Engine API).
- `docs/ai/product.md` §7 — remove the "no proposal detail" gap row.
- `docs/ai/phases/phase-2.md` — completion entry with the final response shapes.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
