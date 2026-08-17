# GRF-201 — Mockup-led design token foundation

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

Implement the mockup-led light SaaS token foundation in the existing Tailwind v4 stylesheet. This ticket is the first child of [GRF-240](GRF-240-mockup-led-studio-product-system.md): it replaces the interim dark/jade theme with the approved off-white, cool-gray, warm-orange visual contract without changing workflow layouts or domain behavior.

## Context

`studio/src/styles.css` already provides the Tailwind v4 entry, shadcn semantic aliases, palette variables, and base layer. It currently expresses the retired dark/jade visual direction. The component library uses Tailwind utilities; the stylesheet remains the only source of raw palette values.

## Scope

### In scope

- Define the mockup-led light palette in the existing `:root` and map it through the existing `@theme inline` aliases.
- Preserve Tailwind v4, shadcn conventions, and `cn()`; do not revive the retired BEM stylesheet split.
- Provide semantic aliases for canvas, panels, overlays, orange primary/focus/selection states, input/border states, and all status tones.
- Establish the global typography, selection, scrollbar, focus-visible, and reduced-motion behavior used by every component.
- Keep raw colour literals centralized in `styles.css`; components consume only Tailwind semantic utilities.

### Out of scope

- New components (GRF-202).
- Layout restructuring (GRF-203).
- New pages or behaviours.

## Acceptance criteria

- [ ] `studio/src/styles.css` remains the single style entry imported by `main.tsx` and contains the only raw palette literals.
- [ ] `:root` provides the light canvas, white card/popover, warm-orange primary/ring, neutral secondary/muted/accent, and semantic status aliases in design-system §2.
- [ ] `@theme inline` exposes each shadcn/Tailwind semantic alias used by Studio components.
- [ ] Components need no raw colour literal or inline style to express normal, hover, disabled, focus, or status states.
- [ ] Body typography, text selection, and scrollbar treatment match the visual contract; no font weight above `700` is introduced.
- [ ] Every existing interactive primitive keeps a visible `focus-visible` treatment through the `ring` aliases; no `outline: none` is introduced without a replacement.
- [ ] The global reduced-motion behavior remains present and correct.
- [ ] Existing routes remain styled at `#ledgers`, `#changes`, `#proposals`, and `#releases`; full responsive layout work remains GRF-203 and the page tickets.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.
- [ ] The app renders with no missing styles at `#ledgers`, `#changes`, `#proposals`, `#releases`.

## Implementation notes

The stack was explicitly revised after this ticket was written: Tailwind v4 `@theme` tokens and shadcn utility composition are authoritative. Do not create a second stylesheet hierarchy or migrate classes to BEM.

## Test plan

- `pnpm test` — existing `client.test.ts` must still pass (no behaviour change).
- Manual: load each of the four routes with the runtime running and confirm no unstyled regions.
- Manual: tab through every control on each page and confirm a visible orange focus ring.
- Manual: enable "Reduce motion" in the OS and confirm transitions stop.

## Docs to update

- `docs/ai/repo-structure.md` — no tree update is required; the Tailwind v4 style entry remains `styles.css`.
- `docs/ai/phases/phase-1.md` — add the completion entry.
- `docs/ai/design-system.md` — mark §2 as implemented; record any token you had to add or rename.

## Definition of done

All acceptance criteria checked, quality gate green, `docs/ai/tickets/INDEX.md` status table updated, phase log entry written.
