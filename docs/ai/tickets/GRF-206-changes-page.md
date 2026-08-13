# GRF-206 — Changes page redesign

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | L |
| Depends on | GRF-203, GRF-204 |
| Blocks | GRF-232 |

## Summary

Rebuild `studio/src/features/changes/changes-page.tsx` into the inbox workspace described in [design-system.md §5.2](../design-system.md): stats strip, filterable data table, multi-select with a sticky action bar, a detail drawer, and a proper submit drawer with JSON validation.

## Context

Current page:

- Left panel: a `.table` of `div` rows showing truncated id, unit, action, and a `StatusBadge`. Not a real table, not sortable, not filterable, not selectable.
- Right panel: a permanent form with `unit` (text) and `desired` (8-row textarea defaulting to `{}`), hardcoded to `action: "PUT"`. `DELETE` cannot be submitted from Studio at all.
- The idempotency key is generated invisibly and cannot be seen or set.
- `JSON.parse` failure produces a generic red line.
- No way to inspect the full desired value of an accepted Change.
- Proposal creation lives on a different page with its own checkbox list, disconnected from this table.

## Scope

### In scope

- Stats strip: `READY`, `RELEASED`, `INVALID` counts.
- Filter bar: status select, action select, unit text search (all client-side over the fetched list).
- `DataTable` with columns: selection, `SEQ`, `UNIT`, `ACTION`, `DESIRED FINGERPRINT`, `STATUS`, `AGE`.
- Sticky selection action bar → `Create proposal` (calls `api.createProposal`, then navigates to `#proposals`).
- Row click → detail `Drawer`.
- `Submit change` `Drawer` supporting both `PUT` and `DELETE`.

### Out of scope

- Server-side pagination and filtering (GRF-214) — filter the fetched page client-side and note the limit.
- Editing or deleting an accepted Change (no API; Changes are immutable by design).
- Bulk import.

## Acceptance criteria

- [ ] `PageHeader` with eyebrow `DURABLE INBOX`, title `Changes`, and the description from design-system §5.2, plus a primary `Submit change` action.
- [ ] Stats strip renders three `Stat` components with live counts derived from the fetched list.
- [ ] `DataTable` is selectable and shows the seven columns above. `UNIT` and `DESIRED FINGERPRINT` use `--font-mono`; the fingerprint uses `HashChip`.
- [ ] `AGE` renders a relative time ("2m ago", "3d ago") with the absolute ISO timestamp in a `title` attribute.
- [ ] Only `READY` rows are selectable. Non-selectable rows render a disabled checkbox with a `title` explaining why.
- [ ] Selecting ≥1 row reveals a sticky bottom bar showing `{n} selected`, a `Clear` action, and a primary `Create proposal` action.
- [ ] `Create proposal` opens a small confirm step capturing the proposal `title`, shows the **ordered** list of selected changes with up/down reordering (the Proposal hash is order-sensitive — see [tech-spec.md §5](../tech-spec.md)), then calls `api.createProposal` and navigates to `#proposals`.
- [ ] A `409` from proposal creation renders the server message verbatim ("One or more Changes are already in another active Proposal.") and refetches the list.
- [ ] Row click opens a `Drawer` showing: id, sequence, unit, action, status, created timestamp, `baseFingerprint`, `desiredFingerprint`, and the full `desired` value in a `CodeBlock`.
- [ ] The detail drawer states plainly when `baseFingerprint` is empty: "Not captured — base fingerprints are recorded from GRF-221 onward."
- [ ] `Submit change` drawer has: `unit` (`Field`, required), an `action` `Segmented` control (`PUT` / `DELETE`), a JSON `Textarea` shown only for `PUT`, and an `idempotencyKey` field pre-filled with a generated value and editable.
- [ ] JSON is validated on blur and on submit. Invalid JSON shows the parser message on the textarea's `Field`, and a `Format` button pretty-prints valid JSON in place.
- [ ] `DELETE` submits with no `desired` payload.
- [ ] Reusing an idempotency key with different content surfaces the server's `409` message verbatim on the key field.
- [ ] A successful submit closes the drawer, refetches, and briefly highlights the new row.
- [ ] All five interaction states implemented; empty state explains that applications post Changes to `POST /api/v1/ledgers/{id}/changes` and shows the curl snippet in a `CodeBlock`.
- [ ] When no ledger is selected, the page renders the "select a ledger" empty state with an action that opens the topbar switcher.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- Relative time: write a small `formatAge(iso: string)` helper in `features/shared/time.ts`. Do not add `date-fns` or `dayjs`.
- Idempotency key default: `studio-{unit}-{Date.now()}` is fine, but it MUST be visible and editable so users understand the guarantee.
- Reordering: keep the selection as an ordered `string[]`, not a `Set` — order is semantically meaningful here.
- Filters are `useState` + `useMemo` over `data.items`. Add a visible note when the list is at the server's unbounded limit, and link the follow-up to GRF-214 in the phase log.
- Do not re-derive which Changes are proposable beyond `status === "READY"`; the server is authoritative and will reject anything else.

## Test plan

- `features/changes/changes-page.test.tsx`:
  - table renders rows from a stubbed response,
  - non-`READY` rows are not selectable,
  - selecting rows reveals the action bar with the right count,
  - invalid JSON blocks submit and shows the parser message,
  - `DELETE` submits without `desired`,
  - `409` on create-proposal renders the server message.
- `features/shared/time.test.ts` for `formatAge`.
- Manual keyboard pass: select rows with `Space`, reach the action bar with `Tab`.

## Docs to update

- `docs/ai/design-system.md` §5.2 — mark implemented, record deviations.
- `docs/ai/product.md` §6 — note that Proposal creation now starts from the Changes page.
- `docs/ai/phases/phase-1.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
