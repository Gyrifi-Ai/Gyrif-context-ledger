# GRF-205 — Ledgers page redesign

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | M |
| Depends on | GRF-203, GRF-204 |
| Blocks | GRF-232 |

## Summary

Rebuild `studio/src/features/ledgers/ledgers-page.tsx` to the card-grid design in [design-system.md §5.1](../design-system.md), translating the dashboard reference defined by [GRF-240](GRF-240-mockup-led-studio-product-system.md) without inventing analytics.

## Context

Current page is a two-column `content-grid`: a left panel listing ledger cards as `<button>` elements, and a right panel permanently occupied by a two-field create form (name only, plus a submit button).

Problems:

- A create form that is used once per month occupies a third of the screen forever.
- Cards show name, description, and a truncated ID — no operational signal (how many changes are waiting? how many releases?).
- The `description` field is accepted by the API but there is no input for it.
- Errors render as bare red text with no retry.
- No loading state; the list flashes empty.
- Selecting a ledger gives no feedback beyond a border colour change.

## Scope

### In scope

- Card grid of ledgers with per-ledger counts.
- Create-ledger flow moved into a `Drawer` triggered by a `+ New ledger` primary action in the `PageHeader`.
- `name` and `description` inputs with validation.
- Active-ledger affordance and selection feedback.
- Full five-state coverage.

### Out of scope

- Deleting or renaming ledgers (no API exists; do not invent one).
- Per-ledger Qdrant collection configuration (collection is process-global — see [product.md §2](../product.md)).

## Acceptance criteria

- [ ] Page uses `PageHeader` with eyebrow `LEDGERS`, title `Ledgers`, and the description from design-system §5.1.
- [ ] Ledgers render in a responsive card grid: 3-up ≥1440 px, 2-up ≥900 px, 1-up below.
- [ ] Each card shows: name (`--text-md --weight-semibold`), description (`--text-sm --text-muted`, clamped to 2 lines), a `HashChip` of the ledger id, and a meta row with `{n} ready` and `{n} releases`.
- [ ] Counts come from `api.changes(id)` filtered to `status === "READY"` and `api.releases(id)`, fetched per visible card with the requests running concurrently. A count that fails renders `—`, never breaks the card.
- [ ] The active ledger card has a `--border-accent` border and an `ACTIVE` `StatusBadge` with the `success` tone.
- [ ] Clicking a card sets the active ledger and shows a 3 s inline confirmation ("Now governing {name}"). It does **not** navigate away.
- [ ] `+ New ledger` opens a `Drawer` with `Field`-wrapped `name` (required, trimmed, max 120 chars) and `description` (optional, max 500 chars) inputs.
- [ ] Submitting with a duplicate name surfaces the server message verbatim ("A ledger with that name already exists.") inline on the name field, not as a page-level error.
- [ ] On success the drawer closes, the list refetches, the new ledger becomes active, and focus returns to the `+ New ledger` button.
- [ ] Loading state renders 3 card skeletons matching the final card dimensions — no layout shift on resolve.
- [ ] Empty state: title "No ledgers yet", description explaining that a ledger is a governed namespace, and a primary `Create your first ledger` action.
- [ ] Error state renders `ErrorState` with a working `Retry`.
- [ ] Full keyboard path: tab to a card, `Enter` selects; tab to `+ New ledger`, `Enter` opens the drawer, `Escape` closes it and restores focus.
- [ ] No raw palette values in the component. Styling uses Tailwind utilities backed by the semantic tokens in `styles.css`; timing constants remain named behavior rather than visual colour data.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- Reuse `useQuery` from GRF-204 for the ledger list and `useMutation` for creation.
- Per-card counts: one `useQuery` per card is acceptable at this scale and keeps failures isolated. Do not build a batching layer.
  - If N grows uncomfortable, note it in the phase log as a candidate for a future `GET /api/v1/ledgers?include=counts` endpoint — do not add the endpoint in this ticket.
- Cards must be real `<button type="button">` elements or `<a>` with a role — never a `div` with `onClick`.
- Use `-webkit-line-clamp: 2` for the description clamp; add a `title` attribute with the full text.
- The inline confirmation should use `role="status"` so screen readers announce it.
- Keep the `AppState.setLedgerId` contract from GRF-203; do not add new global state.

## Test plan

- `features/ledgers/ledgers-page.test.tsx`:
  - renders skeletons then cards,
  - renders `EmptyState` for `{ items: [] }`,
  - renders `ErrorState` and refetches on `Retry`,
  - duplicate-name `409` shows the server message on the name field,
  - selecting a card calls `setLedgerId`.
- Manual: 1440 / 1180 / 900 / 480 px screenshots.

## Docs to update

- `docs/ai/design-system.md` §5.1 — mark implemented, record any deviation.
- `docs/ai/phases/phase-1.md` — completion entry including a before/after description.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
