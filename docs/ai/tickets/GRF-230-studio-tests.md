# GRF-230 — Studio component and integration test suite

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 4 — Qualification |
| Epic | Quality |
| Priority | High |
| Size | L |
| Depends on | GRF-202 |
| Blocks | — |

## Summary

Establish real frontend testing. `studio/src/test/` is an empty directory and the only test in the codebase is `studio/src/api/client.test.ts`.

## Context

Vitest 3.2.4 is installed and `pnpm test` runs, but nothing exercises a component. Every page, every state, and every disabled-action rule in Phase 1 is currently unverified. Given that Phase 1 encodes governance gates in the UI, untested components are a correctness risk, not just a hygiene one.

There is no DOM environment configured and no Testing Library dependency yet.

## Scope

### In scope

- Test environment setup: jsdom, Testing Library, matchers, a shared render helper, and an API mock layer.
- Component tests for every module in `studio/src/ui/`.
- Page-level integration tests for the four feature areas.
- A coverage threshold enforced in CI.

### Out of scope

- Browser end-to-end tests (GRF-232).
- Visual regression / screenshot testing.
- Testing against a live runtime.

## Acceptance criteria

**Setup**

- [x] Dev dependencies added: `jsdom`, `@testing-library/react`, `@testing-library/user-event`, `@testing-library/jest-dom`. These are the only additions permitted; no test-utility grab bag.
- [x] `vite.config.ts` gains a `test` block with `environment: "jsdom"`, `setupFiles: ["src/test/setup.ts"]`, `globals: false` (import from `vitest` explicitly), and `css: true` so token classes resolve.
- [x] `src/test/setup.ts` registers jest-dom matchers, resets handlers between tests, and fails a test on an unexpected `console.error`.
- [x] `src/test/render.tsx` exports a `renderWithProviders` wrapping the component in the app providers, returning the `userEvent` instance.
- [x] `src/test/api-mock.ts` provides a typed `mockApi` built from the `Api` interface so a signature change breaks the mocks at compile time rather than at runtime.
- [x] The mock layer stubs `fetch` — it does not monkey-patch `client.ts` internals.
- [x] `pnpm test` runs the suite headlessly. The script keeps the direct-entry-point form (`node node_modules/vitest/vitest.mjs run`) required by the `:` in the workspace path.

**Component coverage**

- [x] Every component in `src/ui/` has a test file covering: default render, each variant/tone, the disabled state where applicable, and keyboard interaction where applicable.
- [x] `Button` — variants, sizes, `loading` disables and exposes `aria-busy`, `iconOnly` requires an accessible name.
- [x] `StatusBadge` — every domain status maps to the expected tone; the exhaustive mapping is asserted so a new status added to the union fails the test.
- [x] `DataTable` — empty, loading, error, populated; row selection; keyboard row activation.
- [x] `EmptyState`, `ErrorState`, `Skeleton`, `Field`, `HashChip`, `Timeline`, `Stat`, `Panel`, `CodeBlock` — rendered and asserted.
- [x] `ConfirmDialog` — focus is trapped, `Escape` closes, focus returns to the trigger, the confirm action is not fired on dismiss.

**Page integration**

- [x] Each of the four feature pages has a test asserting all five interaction states from design-system §6: loading, empty, populated, error, and a permission/disabled state.
- [x] Proposals: the gate matrix is asserted through the rendered UI, not just the pure function — disabled buttons show their reason text.
- [x] Releases: the rollback confirm dialog contains the four required statements; the recovery banner appears only with intents present.
- [x] Changes: filters, detail drawer, and status rendering.
- [x] Ledgers: creation validation and the switcher.
- [x] Async layer: `useQuery` loading/success/error transitions, `useMutation` in-flight disabling, and `ApiError` code/message propagation into `ErrorState`.

**Discipline**

- [x] No test asserts on CSS class names as a proxy for behaviour. Query by role, label, and text.
- [x] No `waitFor` with an arbitrary sleep. Use Testing Library's async utilities.
- [x] No snapshot tests of whole components. Snapshots of large trees fail noisily and get regenerated without thought.
- [ ] Coverage thresholds: 80% statements and 75% branches across `src/`, enforced by `vitest run --coverage` and wired into CI (GRF-233). Local enforcement is complete; CI wiring remains deferred to GRF-233 because no workflow exists yet.
- [x] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- Add `@vitest/coverage-v8` for coverage; it is the provider matching the installed Vitest major.
- The `mockApi` should be `Record<keyof Api, Mock>` derived from the real interface, so the compiler enforces completeness.
- Prefer `userEvent` over `fireEvent` everywhere — it produces realistic event sequences and catches focus bugs.
- For the exhaustive status mapping test, iterate over a `const` array typed as the status union; TypeScript will reject the array if a member is missing.
- Keep test files adjacent to their subjects (`button.test.tsx` next to `button.tsx`), reserving `src/test/` for shared infrastructure.
- Do not test the design tokens' values. Test that components respond to variants; token values are a design decision, not a behaviour.

## Test plan

This ticket *is* the test plan. Verification is: coverage thresholds met, the suite passes from a clean checkout, and deliberately breaking a gate rule in `proposals` makes a test fail.

## Docs to update

- `docs/ai/tech-spec.md` §12 (test surface) and §13 (quality gate) — record the frontend test setup and the coverage command.
- `docs/ai/repo-structure.md` — `src/test/` contents and the co-located test convention.
- `docs/ai/phases/phase-4.md` — completion entry with the achieved coverage numbers.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
