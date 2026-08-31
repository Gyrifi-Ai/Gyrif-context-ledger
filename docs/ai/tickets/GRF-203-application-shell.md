# GRF-203 — Application shell, navigation, and real runtime status

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | M |
| Depends on | GRF-201, GRF-202 |
| Blocks | GRF-205, GRF-206, GRF-207, GRF-208 |

## Summary

Rebuild the application shell to the layout in [design-system.md §3](../design-system.md): sidebar + sticky topbar + page header, with a ledger switcher, a HEAD chip, and a **real** runtime status indicator. This shell is coordinated by [GRF-240](GRF-240-mockup-led-studio-product-system.md) and follows its light/orange responsive contract.

## Context

`studio/src/ui/layout/application-shell.tsx` currently renders:

- a 240 px sidebar with the brand mark, four nav links, and the literal text `Runtime connected` next to a static green dot — **this is a lie; nothing is probed**;
- a `main > header` with a hardcoded eyebrow `MODULAR MONOLITH` and the raw `ledgerId` in a `<code>` block.

`studio/src/app/router.tsx` is hash-based with a `Route` union of four values. `studio/src/app/providers.tsx` holds `{ ledgerId, setLedgerId }` persisted to `localStorage["gyrifi.ledger"]`.

Problems: no way to switch ledgers except navigating to the Ledgers page; no visibility of `HEAD`; the eyebrow leaks architecture vocabulary into the product; no page description; nav is enabled even when no ledger is selected, producing four identical empty states.

## Scope

### In scope

- Rewrite `ui/layout/application-shell.tsx` as a pure layout component with named slots (`sidebar`, `topbar`, `header`, `children`, `rail`).
- New `features/shell/` containing the domain-aware pieces: `ledger-switcher.tsx`, `head-chip.tsx`, `runtime-status.tsx`, `nav.tsx`.
- New `ui/layout/page-header.tsx` (eyebrow / title / description / actions).
- Extend `app/providers.tsx` with the selected ledger object (not just the id) and a `refreshLedgers` handle.
- Add a `useSystemStatus()` hook polling `GET /api/v1/system/status`.
- Disable Changes/Proposals/Releases nav items when no ledger is selected.

### Out of scope

- Page bodies (GRF-205…208).
- Live event-driven refresh (GRF-210).
- Adding a `HEAD` field to the API — derive it from `api.releases(ledgerId)[0]` for now and note the follow-up.

## Acceptance criteria

- [ ] Shell layout matches design-system §3.1: `--shell-sidebar` sidebar, 56 px sticky topbar with `backdrop-filter`, content region capped at `--shell-max` with `--shell-gutter`.
- [ ] `ApplicationShell` is domain-free — it accepts `ReactNode` slots and imports nothing from `features/` or `api/`.
- [ ] Sidebar nav items render an icon, a label, and an optional count pill, with a 2 px primary-orange left rail on the active item.
- [ ] Nav items other than Ledgers are `aria-disabled`, non-navigable, and show the title "Select a ledger first" when `ledgerId` is empty.
- [ ] Topbar ledger switcher shows the current ledger name, opens a keyboard-navigable popover listing all ledgers with a text filter, and updates `AppState` on selection. Shows `Select ledger` when none is chosen.
- [ ] Topbar HEAD chip renders `HEAD · {releaseId}` via `HashChip`, or `No releases yet`.
- [ ] Runtime status is driven by a real poll of `/api/v1/system/status` every 30 s with these states:
  - 2xx within 2 s ⇒ `Connected` (semantic success),
  - 2xx slower than 2 s, or non-2xx ⇒ `Degraded` (amber),
  - network failure ⇒ `Offline` (rose).
- [ ] Runtime status tooltip/title shows `version` and `inference` from the response.
- [ ] The strings `Runtime connected` and `MODULAR MONOLITH` no longer appear anywhere in `studio/src`.
- [ ] `PageHeader` is used by all four pages with a real one-sentence description each (copy in design-system §5).
- [ ] Polling stops when the document is hidden (`visibilitychange`) and resumes on focus.
- [ ] Responsive: side rail drops below content at 1180 px; sidebar becomes a horizontal scrolling strip at 900 px.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- Keep the hash router. Do **not** add `react-router` — no new runtime dependencies (design-system §8.1).
- `useSystemStatus` belongs in `features/shell/use-system-status.ts`, not in `api/`. `api/client.ts` only gains nothing new — `api.status()` already exists.
- Measure latency with `performance.now()` around the fetch to decide `Degraded`.
- Use `AbortController` in the polling effect cleanup to avoid setState-after-unmount.
- The ledger switcher popover: a `<button aria-haspopup="listbox" aria-expanded>` plus a `role="listbox"` container; `ArrowUp`/`ArrowDown`/`Enter`/`Escape` handled; close on outside click.
- Provider shape suggestion:
  ```ts
  interface AppState {
    ledgerId: string;
    ledger: Ledger | null;
    ledgers: Ledger[];
    setLedgerId: (id: string) => void;
    refreshLedgers: () => Promise<void>;
  }
  ```
  Keep persisting only the id to `localStorage`.

## Test plan

- `features/shell/runtime-status.test.tsx` — stub `fetch` to return 200 / 500 / reject and assert the three visual states.
- `features/shell/ledger-switcher.test.tsx` — keyboard open, filter, select, `Escape` close.
- `ui/layout/application-shell.test.tsx` — renders all slots; no `features/` import.
- Manual: disable the runtime and confirm the indicator turns `Offline` within one poll interval.

## Docs to update

- `docs/ai/repo-structure.md` — add `features/shell/` and the new `ui/layout` files.
- `docs/ai/product.md` §6 — record the ledger-switcher and HEAD chip as part of the product surface; remove the "hardcoded Runtime connected" gap row.
- `docs/ai/tech-spec.md` §11 — document `useSystemStatus` polling behaviour.
- `docs/ai/phases/phase-1.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
