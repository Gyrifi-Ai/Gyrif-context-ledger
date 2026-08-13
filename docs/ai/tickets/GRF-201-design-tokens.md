# GRF-201 — Design token foundation and stylesheet split

| Field | Value |
|---|---|
| Type | Task |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | Highest |
| Size | M |
| Depends on | — |
| Blocks | GRF-202, GRF-203, GRF-204 |

## Summary

Replace the single flat `studio/src/styles.css` with a tokenised, split stylesheet that implements every token in [design-system.md §2](../design-system.md). No visual regression is acceptable, but small improvements that fall out of the token mapping are expected.

## Context

`studio/src/styles.css` is one file with ~280 lines of hardcoded values:

- No CSS custom properties at all. Every colour is a literal hex (`#080b10`, `#6ee7b7`, `#252e3b`, …).
- Arbitrary spacing: `13px`, `9px`, `34px`, `28px`, `7px`, `18px` — no rhythm.
- Font weights `650`, `750`, `800`, `850`, `900`.
- No motion tokens, no `prefers-reduced-motion`, no `:focus-visible` handling (only `:focus`).
- Component styles are keyed on bare element selectors (`aside`, `nav a`, `main > header`, `form`, `label`, `input`) which will collide with any future embedded content.

## Scope

### In scope

- Create `studio/src/styles/` with `tokens.css`, `reset.css`, `base.css`, `components.css`, and an `index.css` that imports them in that order.
- Define every token from design-system §2.1–2.5 verbatim.
- Rewrite existing component styles against tokens, renaming classes to the `gy-{block}__{element}--{modifier}` convention.
- Add the `prefers-reduced-motion` block.
- Replace `:focus` styling with `:focus-visible` using `--focus-ring`.
- Update `studio/src/main.tsx` to import `./styles/index.css`.
- Update every `className` string in `studio/src` to the new names.

### Out of scope

- New components (GRF-202).
- Layout restructuring (GRF-203).
- New pages or behaviours.

## Acceptance criteria

- [ ] `studio/src/styles.css` is deleted; `studio/src/styles/index.css` is the single import in `main.tsx`.
- [ ] `tokens.css` contains only a `:root` block and defines all colour, typography, space, radius, elevation, and motion tokens listed in design-system §2.
- [ ] `grep -nE '#[0-9a-fA-F]{3,8}' studio/src/styles/components.css` returns **zero** matches. Raw hex exists only in `tokens.css`.
- [ ] `grep -rn 'style={{' studio/src` returns no matches except genuinely dynamic values, each with a `// dynamic:` comment explaining why.
- [ ] Every spacing/size declaration in `components.css` uses a `--space-*`, `--radius-*`, or `--shell-*` token.
- [ ] No font weight above `700` remains.
- [ ] Every interactive element has a `:focus-visible` rule applying `--focus-ring`; no `outline: none` without a replacement.
- [ ] `@media (prefers-reduced-motion: reduce)` block present and correct.
- [ ] Responsive breakpoints exist at `1180px`, `900px`, and `480px`.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.
- [ ] The app renders with no missing styles at `#ledgers`, `#changes`, `#proposals`, `#releases`.

## Implementation notes

Suggested file split:

```text
studio/src/styles/
├── index.css        # @import "./tokens.css"; ... in dependency order
├── tokens.css       # :root { --gy-*, --surface-*, --text-*, --space-*, ... }
├── reset.css        # *,*::before,*::after { box-sizing: border-box }, margin resets,
│                    # button/input font inheritance, ::selection, scrollbar styling
├── base.css         # html/body background + type, heading scale, a, code, table defaults
└── components.css   # .gy-* classes only
```

`@import` in a plain CSS file processed by Vite is inlined at build time — no runtime cost. Keep `index.css` as the only entry.

Colour migration map from the current file:

| Old | New token |
|---|---|
| `#080b10` | `--surface-base` |
| `rgba(16,21,29,.86)` | `--surface-raised` |
| `#0b1016` / `#0d1219` | `--surface-inset` / `--surface-sunken` |
| `#202833` / `#252e3b` | `--border-subtle` / `--border-default` |
| `#e7edf4` | `--text-primary` |
| `#718094` | `--text-muted` |
| `#6ee7b7` | `--gy-jade-300` → `--text-accent` |
| `#8df0c8` / `#16382e` | `--status-success-fg` / `--status-success-bg` |
| `#ffadad` / `#3b1f25` | `--status-danger-fg` / `--status-danger-bg` |

Class renames:

| Old | New |
|---|---|
| `.shell` | `.gy-shell` |
| `.panel`, `.panel--wide`, `.panel__heading` | `.gy-panel`, `.gy-panel--wide`, `.gy-panel__header` |
| `.button` | `.gy-button`, `.gy-button--primary`, etc. |
| `.status`, `.status--positive` | `.gy-badge`, `.gy-badge--success` |
| `.empty` | `.gy-empty` |
| `.table`, `.table__row` | `.gy-table`, `.gy-table__row` |
| `.ledger-card` | `.gy-ledger-card` |
| `.timeline` | `.gy-timeline` |
| `.eyebrow` | `.gy-eyebrow` |

Element selectors (`aside`, `nav a`, `main > header`) must become classes.

## Test plan

- `pnpm test` — existing `client.test.ts` must still pass (no behaviour change).
- Manual: load each of the four routes with the runtime running and confirm no unstyled regions.
- Manual: tab through every control on each page and confirm a visible jade focus ring.
- Manual: enable "Reduce motion" in the OS and confirm transitions stop.

## Docs to update

- `docs/ai/repo-structure.md` — replace `styles.css` with the `styles/` tree in the `studio/` listing.
- `docs/ai/phases/phase-1.md` — add the completion entry.
- `docs/ai/design-system.md` — mark §2 as implemented; record any token you had to add or rename.

## Definition of done

All acceptance criteria checked, quality gate green, `docs/ai/tickets/INDEX.md` status table updated, phase log entry written.
