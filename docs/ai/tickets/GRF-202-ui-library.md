# GRF-202 — UI primitive and pattern library

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | Highest |
| Size | L |
| Depends on | GRF-201 |
| Blocks | GRF-203, GRF-204, GRF-205, GRF-206, GRF-207, GRF-208, GRF-230 |

## Summary

Build the reusable, domain-free component library specified in [design-system.md §4](../design-system.md). Today `studio/src/ui/` contains four trivial components with no variants, no states, and no accessibility handling.

## Context

Current `ui/` inventory:

| File | Current implementation |
|---|---|
| `ui/primitives/button.tsx` | `<button className={"button " + className} {...props} />` — no variants, no loading |
| `ui/patterns/status-badge.tsx` | Guesses tone with regex on the label. Domain knowledge leaking into `ui/`. |
| `ui/layout/application-shell.tsx` | Hardcoded nav array, hardcoded "Runtime connected" text |
| `ui/feedback/empty-state.tsx` | `<div><strong>{title}</strong><p>{children}</p></div>` |

There is no table, no panel, no skeleton, no error state, no field wrapper, no dialog, no icon set.

## Scope

### In scope

Create or rewrite:

```text
studio/src/ui/
├── primitives/
│   ├── button.tsx        (rewrite)
│   ├── icon.tsx          (new — inline SVG set)
│   ├── field.tsx         (new — label/control/hint/error wrapper)
│   ├── input.tsx         (new)
│   ├── textarea.tsx      (new)
│   └── segmented.tsx     (new — action PUT/DELETE selector)
├── patterns/
│   ├── status-badge.tsx  (rewrite — tone is a prop)
│   ├── data-table.tsx    (new)
│   ├── hash-chip.tsx     (new)
│   ├── code-block.tsx    (new)
│   ├── timeline.tsx      (new)
│   ├── stat.tsx          (new)
│   └── confirm-dialog.tsx(new)
├── layout/
│   ├── panel.tsx         (new)
│   └── drawer.tsx        (new — right-hand side sheet)
└── feedback/
    ├── empty-state.tsx   (rewrite)
    ├── error-state.tsx   (new)
    └── skeleton.tsx      (new)
```

Also add `studio/src/features/shared/status.ts` holding the **domain→tone** mapping table from design-system §2.2.

### Out of scope

- Wiring components into pages (GRF-205…208).
- Tests beyond a smoke render (full suite is GRF-230, but see acceptance criteria).

## Acceptance criteria

- [ ] Every component's props match the signatures in design-system §4 exactly.
- [ ] `Button` supports `variant` (`primary` | `secondary` | `ghost` | `danger`), `size` (`sm` | `md`), `loading`, `iconLeft`. `loading` sets `aria-busy="true"`, disables the button, and does not change its width.
- [ ] `StatusBadge` takes `tone` as a **prop**. No regex, no string matching, no domain vocabulary anywhere in `ui/`.
- [ ] `features/shared/status.ts` exports `changeTone`, `proposalTone`, `intentTone` covering every enum value in [tech-spec.md §4](../tech-spec.md), with an exhaustive `switch` that fails typecheck if a value is added.
- [ ] `DataTable` renders real `<table>`/`<thead>`/`<tbody>`/`<th scope="col">`, supports selection, row click, sticky header, loading skeletons, and the empty slot.
- [ ] `DataTable` keyboard: `ArrowUp`/`ArrowDown` move row focus, `Space` toggles selection when `selectable`, `Enter` fires `onRowClick`.
- [ ] `ConfirmDialog` uses native `<dialog>`, traps focus, closes on `Escape`, restores focus to the trigger, and does **not** autofocus a destructive button.
- [ ] `Drawer` is a right-hand sheet with the same focus and `Escape` behaviour.
- [ ] `HashChip` truncates to 10 characters plus ellipsis, copies the full value on click via `navigator.clipboard`, and shows a 1.2 s confirmation. Falls back gracefully when the clipboard API is unavailable.
- [ ] `CodeBlock` pretty-prints JSON with 2-space indent, is scrollable with a `maxHeight`, and has a copy button.
- [ ] `Skeleton` honours `prefers-reduced-motion`.
- [ ] `icon.tsx` exports at minimum: `LedgerIcon`, `ChangeIcon`, `ProposalIcon`, `ReleaseIcon`, `CheckIcon`, `AlertIcon`, `CopyIcon`, `ChevronIcon`, `PlusIcon`, `SpinnerIcon`. All 16×16, `stroke="currentColor"`, `stroke-width="1.5"`, `fill="none"`, with `aria-hidden="true"`.
- [ ] No component in `ui/` imports from `features/` or from `api/`.
- [ ] Every new component has a smoke test rendering it in its default and its loading/disabled state.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- Keep components **presentational**. No `useEffect` data fetching, no context reads, no `api` imports.
- Prefer `forwardRef` on `Button`, `Input`, and `Textarea` so `Field` and `Drawer` can manage focus.
- `DataTable` should be generic: `function DataTable<T>({ ... }: DataTableProps<T>)`. Use `getRowId` for keys — never array index.
- For `Field`, generate an id with `useId()` and wire `htmlFor`, `aria-describedby`, and `aria-invalid`.
- `ConfirmDialog` — call `dialogRef.current?.showModal()` in an effect keyed on `open`; `close()` on `Escape` is native, but you must still fire `onClose`.
- Exhaustiveness pattern for `status.ts`:
  ```ts
  function assertNever(value: never): never {
    throw new Error(`Unhandled value: ${String(value)}`);
  }
  ```
- Styles go in `styles/components.css` as `.gy-*` classes. Components receive no inline styles.

## Test plan

- `studio/src/ui/**/*.test.tsx` smoke tests for each component.
- `features/shared/status.test.ts` asserting every enum value maps to the tone in the design-system table.
- Manual keyboard pass on `DataTable` and `ConfirmDialog`.

## Docs to update

- `docs/ai/repo-structure.md` — new `ui/` and `features/shared/` file listing.
- `docs/ai/design-system.md` — mark §4 implemented; record any prop signature that had to change and why.
- `docs/ai/phases/phase-1.md` — completion entry listing every component built and its final props.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written with the final component inventory.
