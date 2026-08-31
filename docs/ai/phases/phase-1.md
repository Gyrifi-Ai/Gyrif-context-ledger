# Phase 1 — Studio experience

**Goal:** turn the functional-but-unfinished Studio into a mockup-led, light SaaS interface that makes the governance model obvious. The runtime already enforces the rules; Phase 1 makes them visible, explicable, and hard to get wrong.

**Status:** Complete

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-240](../tickets/GRF-240-mockup-led-studio-product-system.md) | Mockup-led Studio product system | XL | — | Done |
| [GRF-201](../tickets/GRF-201-design-tokens.md) | Mockup-led design token foundation | M | — | Done |
| [GRF-202](../tickets/GRF-202-ui-library.md) | UI primitive and pattern library | L | GRF-201 | Done |
| [GRF-203](../tickets/GRF-203-application-shell.md) | Application shell, navigation, real runtime status | M | GRF-202 | Done |
| [GRF-204](../tickets/GRF-204-async-data-layer.md) | Async data layer | M | — | Done |
| [GRF-205](../tickets/GRF-205-ledgers-page.md) | Ledgers page redesign | M | GRF-203, GRF-204 | Done |
| [GRF-206](../tickets/GRF-206-changes-page.md) | Changes inbox redesign | L | GRF-203, GRF-204 | Done |
| [GRF-207](../tickets/GRF-207-proposals-workspace.md) | Proposals review workspace | XL | GRF-203, GRF-204, GRF-211 | Done |
| [GRF-208](../tickets/GRF-208-releases-timeline.md) | Releases timeline and rollback flow | L | GRF-203, GRF-204, GRF-213 | Done |
| [GRF-209](../tickets/GRF-209-studio-resilience.md) | Studio resilience: error boundary, offline state, stream reconnection | M | GRF-202, GRF-204 | Done |

## Phase-level notes

- **Two tickets reach into Phase 2.** GRF-207 needs GRF-211 (proposal detail and evidence reads) and GRF-208 needs GRF-213 (release intent inspection). Do not fake that data client-side to unblock the UI — a governance surface that displays locally-remembered state is worse than one that displays nothing.
- **Do GRF-209 early**, right after the component library. Every screen built before it inherits a shell that fails with a white page, and retrofitting error boundaries is harder than building on top of them.
- The full [design-system.md](../design-system.md) is normative for this phase. Deviations from it are deviations from the ticket and must be logged.
- No frontend dependencies beyond the shadcn/Tailwind set authorised on 2026-08-17 are permitted in this phase; see design-system §8.

## Exit criteria

- [x] All nine tickets complete.
- [x] Every page implements all five interaction states from design-system §6.
- [x] The full workflow — create ledger, review changes, propose, evaluate, approve, release, roll back — is completable in the browser without reading API documentation.
- [x] Every disabled action states its reason on screen.
- [x] Keyboard-only operation of the entire workflow is possible.
- [x] No failure is silent: a render error, an unreachable runtime, and a dropped event stream each produce a visible, recoverable state.
- [x] `pnpm typecheck && pnpm test && pnpm build` green.

## Completed entries

### Un-ticketed — Studio visual redesign on shadcn/ui + Tailwind CSS v4

| | |
|---|---|
| Completed | 2026-08-17 |
| Commit / PR | — |
| Deviated from ticket | N/A — no ticket; owner-directed redesign |

**What was built**

The pre-Phase-1 Studio scaffold (one flat `styles.css` of hardcoded hex, regex-guessed status tones, hardcoded "Runtime connected") was rebuilt as a shadcn/ui application. The owner explicitly authorised the dependency addition after being shown the conflict with the repo's no-dependency rule. All four pages (Ledgers, Changes, Proposals, Releases) were re-skinned with real shadcn components; no new product features were added.

**Files added**

- `studio/src/lib/utils.ts` — `cn()` (clsx + tailwind-merge)
- `studio/src/components/ui/` — shadcn components: `button`, `card`, `badge`, `input`, `textarea`, `label`, `table`, `dialog`, `checkbox`, `separator`, `skeleton`, `tooltip`
- `studio/src/features/shared/status.ts` — `statusTone(value: string): StatusTone`, the normative domain→tone mapping (design-system §2.2)

**Files changed**

- `studio/src/styles.css` — now the Tailwind v4 entry: `@import "tailwindcss"`, `tw-animate-css`, the §2 palette as `@theme` tokens, base layer
- `studio/src/ui/layout/application-shell.tsx` — lucide nav icons, jade active rail, real 30 s `GET /api/v1/system/status` probe with tooltip (replaces hardcoded "Runtime connected"), per-route page headers
- `studio/src/features/{ledgers,changes,proposals,releases}/*-page.tsx` — rebuilt on shadcn Card/Table/Dialog/Checkbox; ledger creation moved into a Radix dialog; Release/Rollback use the destructive variant; list reads tolerate `items: null`
- `studio/src/ui/patterns/status-badge.tsx` — renders `Badge` with `statusTone()`; regex guessing removed
- `studio/src/ui/feedback/empty-state.tsx` — Tailwind styling
- `studio/vite.config.ts` — `@tailwindcss/vite` plugin, `@` → `src` alias, proxy moved to `127.0.0.1:18080` (see traps)
- `studio/tsconfig.json` — `baseUrl` + `paths` for `@/*`
- `studio/package.json` / `pnpm-lock.yaml` — new dependency set
- `studio/index.html` — theme-color `#06080c`
- `docs/ai/design-system.md` §8, `docs/ai/repo-structure.md` §3, `AGENTS.md` §3 — dependency rule revised

**Files removed**

- `studio/src/ui/primitives/button.tsx` — superseded by `components/ui/button.tsx`

**Contracts introduced or changed**

- `cn(...inputs: ClassValue[]): string` in `src/lib/utils.ts`
- `statusTone(value: string): "neutral" | "info" | "review" | "success" | "warning" | "danger"` in `src/features/shared/status.ts`
- New Studio dependencies: `tailwindcss`, `@tailwindcss/vite`, `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-react`, `@radix-ui/react-{dialog,checkbox,label,select,separator,tooltip}`, dev: `tw-animate-css`

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Full shadcn/ui + Tailwind v4 | Owner chose it explicitly when shown the dependency-rule conflict | Hand-rolled CSS on the token system | Owner judged the prior result not shippable |
| Tailwind v4 (`@tailwindcss/vite`, `@theme` in CSS) | Current shadcn convention; no `tailwind.config.js` needed | Tailwind v3 + PostCSS | Legacy path |
| Runtime on port 18080 for local dev | Port 8080 is occupied by an unrelated Java service on this machine | Kill the other process | Not ours to kill |

**Deviations from the ticket**

No ticket existed. The work deliberately did **not** implement GRF-201…209 acceptance criteria (loading skeletons on every surface, error states with retry, keyboard-navigable DataTable, proposal workspace, etc.) — those tickets remain open and are now unblocked on the component library.

**Traps for future work**

- **Latent runtime bug:** `ListChanges` fails with `INTERNAL "Could not load Changes."` when any row has `action='DELETE'` — `desired` is a NULL BLOB and `scanChange` scans it into `json.RawMessage` (non-nullable). Reproduced live; the row set loads fine once DELETE rows are excluded. Needs a ticket (scan into `sql.NullString`/`[]byte` or store `[]byte{}` for deletes).
- **Latent API wart:** empty lists serialise as `"items": null` (Go nil slice), which crashed pages on `items.length`. Pages now coerce with `?? []`; the cleaner fix is server-side (`[]T{}` initialisation).
- Vite hash-navigation does not refetch; pages only load on mount. GRF-204's data layer should own refetch-on-route.
- The dev proxy now targets `127.0.0.1:18080`; the runtime must be started with `GYRIFI_HTTP_ADDRESS=127.0.0.1:18080` locally.

**Tests added**

- None — no behaviour changed; existing `src/api/client.test.ts` still passes. Component tests are GRF-230 scope.

**Docs updated**

- `docs/ai/design-system.md` §8 — stack authorised, token location moved to `@theme`, BEM rule retired
- `docs/ai/repo-structure.md` §3 — new tree, stack description, proxy port
- `AGENTS.md` §3 — dependency rule now names the authorised shadcn set

**Verification**

```
$ pnpm typecheck   → clean (tsc -b, strict)
$ pnpm test        → Test Files 1 passed (1), Tests 2 passed (2)
$ pnpm build       → dist/index.html 0.45 kB, index-*.css 33.05 kB, index-*.js 325.80 kB, ✓ built in 1.07s
```

Visual verification: all four pages exercised in the browser against a live runtime with seeded data (2 ledgers, 2 changes, 1 proposal); dialog focus trap, active-ledger state, and 480 px responsive layout confirmed.

**Follow-ups discovered**

- Ticket needed: `ListChanges` NULL-desired scan failure for DELETE changes (see traps).
- Ticket needed (or fold into GRF-204): server should return `"items": []` instead of `null` for empty lists.
- GRF-201/202/203 are partially pre-empted by this change; their acceptance criteria should be re-scoped against `components/ui/` when picked up.

### GRF-202 — UI primitive and pattern library; GRF-203 — Application shell

| | |
|---|---|
| Completed | 2026-08-17 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

Implemented the domain-free Studio component library: accessible controls, field wrappers, table, feedback states, panels, drawers, confirmation dialogs, copyable hashes/JSON, timeline, stats, and inline SVG icons. Rebuilt the application shell as a slot-based layout, then added domain-aware navigation, selected-ledger switching, HEAD display, and a real 30-second runtime probe. The side navigation prevents entry to ledger-scoped pages until a Ledger is selected.

**Files added**

- `studio/src/ui/primitives/`, `studio/src/ui/patterns/`, `studio/src/ui/feedback/`, `studio/src/ui/layout/` — GRF-202 component library
- `studio/src/features/shell/` — ledger switcher, HEAD chip, nav, status display, and polling hook
- `studio/src/ui/ui-smoke.test.tsx` and `studio/src/features/shared/status.test.ts` — component and status-mapping smoke coverage

**Files changed**

- `studio/src/app/providers.tsx` — selected Ledger object, ledger list, and refresh handle
- `studio/src/app/shell.tsx` / `studio/src/ui/layout/application-shell.tsx` — domain-aware composition around the slot-only layout
- `studio/src/api/client.ts` / `studio/src/api/types.ts` — request status classification and typed lifecycle values
- `studio/src/features/{ledgers,changes,proposals,releases}/` — explicit empty state and caller-owned status tone usage
- `studio/src/styles.css` — shell geometry tokens

**Files removed**

None.

**Contracts introduced or changed**

```ts
type AppState = { ledgerId: string; ledger: Ledger | null; ledgers: Ledger[]; setLedgerId(id: string): void; refreshLedgers(): Promise<void> };
function ApplicationShell({ sidebar, topbar, header, children, rail }: { sidebar: ReactNode; topbar: ReactNode; header: ReactNode; children: ReactNode; rail?: ReactNode }): ReactNode;
function changeTone(status: Change["status"]): StatusTone;
function proposalTone(status: Proposal["status"]): StatusTone;
function intentTone(status: ReleaseIntentStatus): StatusTone;
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep `ui/` domain-free | Prevents browser vocabulary from re-deriving governance semantics | Status regex in badge component | It drifts when server enums change. |
| Classify HTTP failure as degraded and transport failure as offline | Operators can distinguish a responding but unhealthy runtime from an unreachable one | Treat all failed polls as offline | It hides a useful operational distinction. |
| Use the newest Release as temporary HEAD | The API has no Head endpoint yet | Add a client-side governance model | The server remains authoritative. |

**Deviations from the ticket**

None. Native `<dialog>` supplies its modal focus containment; the destructive confirmation action is never autofocusable.

**Traps for future work**

- `api.status()` must retain `RequestInit` support and HTTP `status` metadata so the runtime indicator can distinguish degradation from a network outage.
- HEAD is derived from the newest Release until an explicit API contract is added; do not independently calculate governance readiness in Studio.
- The shell uses the mobile horizontal navigation strip below the desktop breakpoint; optional side rails are supplied through the domain-free `rail` slot.

**Tests added**

- `studio/src/ui/ui-smoke.test.tsx` — renders every primitive, pattern, layout, and feedback component, including loading/disabled controls
- `studio/src/features/shared/status.test.ts` — protects exhaustive normative lifecycle tone mappings

**Docs updated**

- `docs/ai/design-system.md` §4 — component specification implementation status
- `docs/ai/product.md` §6–§7 — selected Ledger/HEAD/status product surface and closed hardcoded-status gap
- `docs/ai/tech-spec.md` §11 — expanded AppState and runtime polling contract
- `docs/ai/repo-structure.md` §3 — current Studio file tree

**Verification**

```
$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Test Files  3 passed (3)
Tests  4 passed (4)
✓ 1899 modules transformed.
✓ built in 1.05s
```

**Follow-ups discovered**

GRF-204 must consolidate per-page list fetching into its async state layer. The shell intentionally owns only Ledger selection and runtime health.

### GRF-201 — Mockup-led design token foundation

| | |
|---|---|
| Completed | 2026-08-17 |
| Commit / PR | — |
| Deviated from ticket | No |

**What was built**

Studio now has the approved light SaaS token foundation: an off-white application canvas, white working surfaces, cool-gray navigation and borders, a warm-orange primary/focus/selection colour, and distinct semantic status colours. The existing Tailwind v4 and shadcn implementation remains in place; this ticket changes no route, API call, workflow, or domain decision.

**Files added**

- None.

**Files changed**

- `studio/src/styles.css` — replaced interim dark/jade values with raw and semantic light-theme tokens, mapped them to Tailwind aliases, and added global focus and reduced-motion behavior.
- `docs/ai/design-system.md` — established the designer mockups as the visual reference and documented the implemented token source.
- `docs/ai/tickets/GRF-201-design-tokens.md` — re-scoped the obsolete stylesheet-split work order to the current Tailwind v4 implementation.
- `docs/ai/tickets/INDEX.md` — recorded GRF-201 completion and the GRF-240 umbrella ticket.

**Files removed**

- None.

**Contracts introduced or changed**

```css
:root {
	--surface-base: var(--gy-slate-050);
	--surface-raised: var(--gy-white);
	--action-primary-bg: var(--gy-orange-500);
	--focus-ring: 0 0 0 2px var(--surface-raised), 0 0 0 4px var(--gy-orange-400);
}
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep one Tailwind v4 style entry | The installed stack already centralizes shadcn theme aliases in `styles.css` | Recreate the retired split BEM stylesheet plan | It conflicts with the authorised Tailwind/shadcn conventions and would duplicate the styling system. |
| Warm orange is the primary accent | It matches the approved mockups and clearly separates brand/selection from semantic success | Retain jade from the interim redesign | It no longer matches the final designer direction. |
| Keep semantic green, amber, and rose independent of brand | Governance states must not be confused with selection or navigation | Use orange for all positive/current states | Orange is brand and selection, not evidence of safe release state. |

**Deviations from the ticket**

None. The ticket was re-scoped before implementation because its original BEM stylesheet split had already been superseded by the owner-authorised Tailwind v4 architecture.

**Traps for future work**

- `@theme` font variables must not refer to themselves. Keep the root `--font-sans` and `--font-mono` aliases as the source used by Tailwind utility classes.
- Existing page layouts intentionally remain unchanged in this ticket. The visual foundation is not a substitute for the shell, component, async, and workflow tickets.
- The mockups are visual-only references. Do not add dashboard, system, sharing, avatar, trend, or CRM surfaces without a product/API decision.

**Tests added**

- None — token values have no behavioral unit-test surface. Existing client tests and the production build validate that the theme compiles without changing API behavior.

**Docs updated**

- `docs/ai/design-system.md` §0–§3 — mockup-led visual direction, light tokens, and orange interaction language.
- `docs/ai/tickets/GRF-201-design-tokens.md` — current implementation scope and acceptance criteria.
- `docs/ai/tickets/INDEX.md` — GRF-201 status and GRF-240 registration.

**Verification**

```
$ pnpm typecheck
$ pnpm test
Test Files  1 passed (1)
Tests       2 passed (2)
$ pnpm build
✓ 1902 modules transformed.
✓ built in 1.03s

$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference      1.538s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger         2.907s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 2.031s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                    2.692s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Already up to date
Test Files  1 passed (1)
Tests       2 passed (2)
✓ 1902 modules transformed.
✓ built in 1.01s

$ cd .. && docker build -t gyrifi:local .
[+] Building 83.8s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <(ls GRF-*.md | grep -oE 'GRF-[0-9]+' | sort) <(grep -oE '^\| GRF-[0-9]+ \| (Not started|In progress|Done)' INDEX.md | grep -oE 'GRF-[0-9]+' | sort)
tickets consistent

Manual browser verification: `#ledgers` rendered at 1440 × 1024 px against a live local runtime. The off-white canvas, orange primary action/focus/selection styling, white cards, light sidebar, and semantic connected status rendered without unstyled regions.
```

**Follow-ups discovered**

- GRF-202 must provide the mockup-specific reusable primitives — grouped navigation, dense DataTable, detail drawer, KPI strip, and floating selection bar — before the shell and workflow pages are rebuilt.
- The known DELETE `desired = NULL` runtime scan failure remains a server bug; it needs its own Phase 2 ticket before Changes can reliably render DELETE rows.

### GRF-204 — Async data layer with loading, error, and empty states

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

Studio now has a small dependency-free async state layer. The four feature pages preserve their current layouts but now render loading skeletons, retryable server errors, explicit empty states, and dimmed stale content during refetches. Every mutating action has an in-flight guard, disables itself while pending, and surfaces a server error instead of swallowing it.

**Files added**

- `studio/src/app/use-async.ts` — abort-safe `useQuery` and guarded `useMutation` hooks.
- `studio/src/app/use-ledger-events.ts` — feature-flagged EventSource query invalidation hook for GRF-210 domain events.
- `studio/src/ui/feedback/async-boundary.tsx` — loading/error/empty/populated query state renderer.
- `studio/src/ui/feedback/async-boundary.test.tsx` — rendered coverage of all query boundary branches.
- `studio/src/vite-env.d.ts` — Vite environment declarations for the event feature flag.

**Files changed**

- `studio/src/api/client.ts` — typed `ApiError`, resilient error-envelope parsing, and signal-capable list reads.
- `studio/src/api/client.test.ts` — structured and malformed error-envelope coverage.
- `studio/src/app/providers.tsx` — ledger context list now uses the query primitive rather than swallowing its initial fetch failure.
- `studio/src/components/ui/button.tsx` — added the existing Studio `loading` control state for mutation buttons.
- `studio/src/features/{ledgers,changes,proposals,releases}/*-page.tsx` — query boundaries, mutation state, pending buttons, retryable failures, and event invalidation wiring.
- `studio/src/features/shell/head-chip.tsx` — abort-safe releases query and visible HEAD read failure.
- `studio/src/styles.css` — `gy-is-refetching` stale-content opacity class.
- `docs/ai/tech-spec.md`, `docs/ai/repo-structure.md`, `docs/ai/tickets/INDEX.md` — current contracts, source tree, and ticket status.

**Files removed**

None.

**Contracts introduced or changed**

```ts
export class ApiError extends Error {
	constructor(readonly code: string, message: string, readonly status: number);
}

export function useQuery<T>(
	key: string,
	fn: (signal: AbortSignal) => Promise<T>,
	deps: unknown[],
): QueryResult<T>;

export function useMutation<TArgs, TResult>(
	fn: (args: TArgs) => Promise<TResult>,
): MutationResult<TArgs, TResult>;

export function useLedgerEvents(onInvalidate: () => void): void;
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep the hooks cache-free | The ticket permits only active-request lifecycle management at this scale | Add a query library or global cache | New dependencies and cross-screen state are explicitly out of scope |
| Abort and replace a query for every refetch | Dependency changes and retries cannot commit stale responses | Let overlapping responses race | Older results could overwrite the current ledger data |
| Gate EventSource invalidation on `VITE_GYRIFI_ENABLE_LEDGER_EVENTS` | The current server event is a connection handshake, not a domain update | Connect every screen now | It produces unnecessary refetches until GRF-210 emits real ledger events |

**Deviations from the ticket**

None. The requested event hook is implemented behind its requested disabled flag; setting `VITE_GYRIFI_ENABLE_LEDGER_EVENTS=true` connects it and invalidates on each `ledger` event.

**Traps for future work**

- `useQuery` callback identity is deliberately not a dependency: callers must list the data inputs in `deps`, and the hook reads the latest callback through a ref.
- A query's `refetch` aborts the prior request, preserves already-resolved data, and dims it through `gy-is-refetching`; do not replace that content with a spinner.
- `useMutation.run` captures errors into mutation state rather than rejecting. Trigger it with `void`, then render `error` near the relevant action and call `refetch` only after a confirmed successful request.
- Keep the ledger event flag disabled until GRF-210 replaces the server's connection `ledger` event with domain invalidations.

**Tests added**

- `studio/src/api/client.test.ts` — confirms `ApiError` preserves a `CONFLICT` envelope and safely labels malformed responses `UNKNOWN`.
- `studio/src/ui/feedback/async-boundary.test.tsx` — verifies loading skeleton, retryable error, empty, populated, and stale-content class rendering.

**Docs updated**

- `docs/ai/tech-spec.md` §11 — API error, async-hook, boundary, EventSource-flag, and 18080 proxy contracts.
- `docs/ai/repo-structure.md` §3 — new app infrastructure and feedback boundary files.
- `docs/ai/tickets/INDEX.md` — GRF-204 completion.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference      1.622s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger         2.605s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 2.139s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                    3.334s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  4 passed (4)
Tests       7 passed (7)
✓ 1905 modules transformed.
✓ built in 1.01s

$ cd .. && docker build -t gyrifi:local .
ERROR: failed to solve: failed to fetch anonymous token from auth.docker.io: connection reset by peer
```

**Follow-ups discovered**

- The Docker image build was blocked by an external Docker Hub authentication connection reset before any build stage ran. Re-run the image portion of the quality gate once registry access is available.

### GRF-204 — Post-completion StrictMode audit correction

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | No |

**What was built**

The approval audit found that `useMutation` left its mounted guard false after React StrictMode's development setup/cleanup/setup cycle. Mutation requests still ran, but their pending, success, and error state could not update. The effect now restores the mounted guard during every setup, and a live StrictMode ledger creation confirmed that mutation state, dialog closure, refetch, active-ledger selection, and focus restoration complete normally.

**Files added**

None.

**Files changed**

- `studio/src/app/use-async.ts` — restore the mutation mounted guard during each effect setup.
- `docs/ai/phases/phase-1.md` — record the audit correction and replace the stale phase status.

**Files removed**

None.

**Contracts introduced or changed**

None. The existing `useMutation<TArgs, TResult>(fn): MutationResult<TArgs, TResult>` contract is unchanged.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Reset `mountedRef.current` in the effect setup | React StrictMode deliberately replays effect setup and cleanup in development | Remove the mounted guard | Async completion could update state after a real unmount |
| Keep this as a focused correction | The rest of the GRF-204 implementation and documentation matched the ticket | Reimplement the async layer | It would add risk and exceed the audit scope |

**Deviations from the ticket**

None.

**Traps for future work**

An effect-owned mounted guard must be set in the effect setup, not only in the ref initializer. Initializers do not run again during StrictMode's effect replay.

**Tests added**

None. GRF-230 adds the DOM hook harness required for an automated StrictMode lifecycle regression test. This correction was exercised manually through the current StrictMode root against a live runtime.

**Docs updated**

- `docs/ai/phases/phase-1.md` — correction, verification evidence, and current phase status.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference       (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant  (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                    (cached)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Already up to date
Test Files  4 passed (4)
Tests       7 passed (7)
✓ 1905 modules transformed.
✓ built in 1.13s

$ docker build -t gyrifi:local .
[+] Building 62.0s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

Manual browser verification: with the React StrictMode root active, creating `grf-204-audit` closed the dialog, returned focus to `New ledger`, refetched the list from two to three Ledgers, selected the new Ledger, and enabled the ledger-scoped navigation.

**Follow-ups discovered**

- GRF-230 should include an automated StrictMode replay regression for `useMutation` once its jsdom and Testing Library harness lands.

### GRF-209 — Studio resilience: error boundary, offline state, stream reconnection

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | No |

**What was built**

Studio now distinguishes an unreachable Runtime from a reachable HTTP or target failure. Transport failures preserve and dim stale query data, disable every mutation with a visible global reason, and trigger visibility-aware status probes with bounded backoff. Root and route-section error boundaries replace white-screen failures with resettable error surfaces, while the event stream now reports connection state, retries permanently closed sources with jitter and a ceiling, offers manual reconnection, and refetches the active page after reconnect.

**Files added**

- `studio/src/app/error-boundary.tsx` — resettable class boundary with once-per-error logging.
- `studio/src/app/error-boundary.test.tsx` — fallback/reset and duplicate-log regression coverage.
- `studio/src/app/reachability.tsx` — shared Runtime polling, request-health, stream-state, and invalidation provider.
- `studio/src/app/reachability-banner.tsx` — persistent application-level transport-failure banner.
- `studio/src/app/reachability.test.ts` — exact 1-to-30-second backoff coverage.
- `studio/src/api/events.test.ts` — stream transition, retry-ceiling, manual-reconnect, and teardown coverage.

**Files changed**

- `studio/src/api/client.ts` / `client.test.ts` — transport/HTTP discrimination, request IDs, and request-health events.
- `studio/src/api/events.ts` — observable EventSource lifecycle with bounded explicit retry for `CLOSED` sources.
- `studio/src/app/bootstrap.tsx` — root boundary and single-owner caught-error logging.
- `studio/src/app/providers.tsx` — compose reachability outside AppState.
- `studio/src/app/shell.tsx` — global banner and route-section boundary.
- `studio/src/app/use-async.ts` — unavailable query state and reachability-gated mutations.
- `studio/src/app/use-ledger-events.ts` — register active-page reconnect invalidation with the shared stream.
- `studio/src/features/{ledgers,changes,proposals,releases}/*-page.tsx` — disable current mutation controls and retries while offline.
- `studio/src/features/shell/runtime-status.tsx` — combined HTTP/stream state and manual reconnect control.
- `studio/src/ui/feedback/async-boundary.tsx` / `async-boundary.test.tsx` — preserve and dim unavailable data.
- `studio/src/ui/feedback/error-state.tsx` — caller-owned action label and disabled reason.
- `studio/src/ui/layout/application-shell.tsx` — application-level banner slot.
- `docs/ai/design-system.md`, `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/tickets/INDEX.md` — current resilience contract, tree, closed gap, and status.

**Files removed**

- `studio/src/features/shell/use-system-status.ts` — superseded by the shared reachability provider; separate pollers could disagree about whether the Runtime was reachable.

**Contracts introduced or changed**

```ts
type ApiErrorKind = "transport" | "http";

class ApiError extends Error {
	constructor(code: string, message: string, status: number, kind: ApiErrorKind, requestId?: string);
}

function subscribeToRequestHealth(
	listener: (health: { reachable: true } | { reachable: false; error: ApiError }) => void,
): () => void;

type QueryResult<T> = {
	data: T | undefined;
	error: Error | undefined;
	loading: boolean;
	refetching: boolean;
	unavailable: boolean;
	refetch: () => void;
};

type MutationResult<TArgs, TResult> = {
	run: (args: TArgs) => Promise<void>;
	pending: boolean;
	blocked: boolean;
	disabledReason: string | undefined;
	error: Error | undefined;
	result: TResult | undefined;
	reset: () => void;
};

type EventStreamState = "connecting" | "open" | "closed";
type EventSubscription = { close(): void; reconnect(): void };

function subscribeToEvents(options: {
	onState(state: EventStreamState): void;
	onReconnect(): void;
	onExhausted(exhausted: boolean): void;
}): EventSubscription;
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| HTTP responses always mean the Runtime is reachable | A `503` release response describes target failure, not loss of the Runtime | Treat every non-2xx as offline | It sends operators to the wrong diagnosis and hides recovery-required state |
| Keep transport failure out of page `error` state | Reachability is application-level and stale data remains useful | Replace every page with `ErrorState` | It duplicates the banner and removes known-good governance data |
| Use one provider for polling and stream state | Banner, topbar, mutation gates, and reconnect invalidation must agree | Keep the old per-component status hook | Independent pollers race and can display contradictory states |
| Let EventSource handle `CONNECTING`; explicitly replace only `CLOSED` | Browser retry semantics are correct for transient transport loss | Always close and recreate on `onerror` | It creates competing reconnect loops |
| Suppress React's root caught-error logger and log in the boundary | Guarantees one explicit message with component stack and one `onError` call | Keep both loggers | Every caught render error appeared twice in the console |

**Deviations from the ticket**

None. The reconnect ceiling is six explicit attempts. Stream delays use ±20% jitter over exponential 1, 2, 4, 8, 16, and 30-second bases; reachability probes use exact exponential delays capped at 30 seconds.

**Traps for future work**

- Abort errors must pass through without marking the Runtime offline; StrictMode and dependency changes intentionally abort reads.
- The Vite development proxy converts an unavailable upstream into an HTTP 500. That correctly remains an HTTP failure under the client contract; transport behavior was browser-verified by aborting requests below the proxy.
- `EventSource` may remain `CONNECTING` indefinitely under native retry. Explicit retry and the attempt ceiling apply only after its `readyState` becomes `CLOSED`.
- GRF-210 must extend the shared stream subscription rather than creating one EventSource per page.

**Tests added**

- `studio/src/api/events.test.ts` — `connecting → open → closed → open`, bounded retry, manual reconnect, source closure, and timer cleanup.
- `studio/src/app/error-boundary.test.tsx` — fallback/reset contract and one log/`onError` call per captured error.
- `studio/src/app/reachability.test.ts` — 1, 2, 4, 8, 16, 30, 30-second probe delay sequence.
- `studio/src/api/client.test.ts` additions — rejected-fetch transport error, request-health recovery, request ID, and reachable HTTP 503.

**Docs updated**

- `docs/ai/design-system.md` §4.6 / §6 — boundary use and application-level unreachable state.
- `docs/ai/product.md` §7 — removed the render/stream silent-failure gap.
- `docs/ai/repo-structure.md` §3 — resilience files and provider ownership.
- `docs/ai/tech-spec.md` §11 / §12 / §14 — exact contracts, tests, and closed technical gap.
- `docs/ai/tickets/INDEX.md` — GRF-209 marked Done.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference       (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant  (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                    (cached)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  7 passed (7)
Tests       16 passed (16)
✓ 1908 modules transformed.
✓ built in 1.08s

$ docker build -t gyrifi:local .
[+] Building 33.1s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

Manual browser verification against a live Runtime: the stream and HTTP probe reached `Connected`; an intercepted transport failure displayed the persistent banner, marked the topbar `Offline`, retained the page shell, and disabled `Create ledger` with the exact banner message in its title. Removing the failure caused a successful request to clear the banner automatically and re-enable the mutation. A real Runtime stop behind Vite produced an HTTP 500 rather than a false transport classification, and restart recovered to `Connected` without reloading Studio.

**Follow-ups discovered**

- GRF-230 should add jsdom integration coverage for banner rendering, visibility changes, and mutation-button gating through the rendered provider tree.
- GRF-210 must add domain-event parsing and dispatch to the existing shared stream; reconnect invalidation is already wired.

### GRF-205 — Ledgers page redesign

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | No |

**What was built**

The Ledgers page changed from a two-column card/form surface with no operational signal into a responsive governance index. Every visible ledger now has an isolated READY-Change count, Release count, copyable ID, active marker, and keyboard-native selection; creation lives in a right-hand drawer with both API fields, boundary validation, exact duplicate-name feedback, and deterministic focus restoration. Loading, empty, HTTP error, unavailable, refetching, and populated states are all explicit.

**Files added**

- `studio/src/features/ledgers/ledgers-page.logic.ts` — name/description limits, boundary validation, and READY count helper kept outside the Fast Refresh component module.
- `studio/src/features/ledgers/ledgers-page.test.tsx` — rendering-state, count fallback, conflict placement, validation, and READY-count regressions.

**Files changed**

- `studio/src/features/ledgers/ledgers-page.tsx` — responsive cards, per-card count reads, active selection, inline confirmation, and create drawer.
- `studio/src/app/shell.tsx` — lets the Ledgers page own its `PageHeader` so the primary action occupies the header action slot; other route headers are unchanged.
- `studio/src/styles.css` — exact 900/1440 viewport grid breakpoints and a Tailwind alias for the semantic accent-border token.
- `studio/src/ui/layout/drawer.tsx` — explicit Escape key close path for native-dialog consistency.
- `docs/ai/design-system.md`, `docs/ai/tickets/INDEX.md` — implemented design and ticket status.

**Files removed**

None.

**Contracts introduced or changed**

No HTTP or global AppState contract changed. The page continues to consume:

```ts
api.ledgers(init?: RequestInit): Promise<{ items: Ledger[] }>;
api.changes(ledgerId: string, init?: RequestInit): Promise<{ items: Change[] }>;
api.releases(ledgerId: string, init?: RequestInit): Promise<{ items: Release[] }>;
api.createLedger(input: { name: string; description?: string }): Promise<Ledger>;

AppState.setLedgerId(id: string): void;
AppState.refreshLedgers(): Promise<void>;
```

Local page helpers introduced for testable boundary behavior:

```ts
const ledgerNameMaxLength = 120;
const ledgerDescriptionMaxLength = 500;
function validateLedgerForm(name: string, description: string): { name?: string; description?: string };
function countReadyChanges(changes: Change[]): number;
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Each `LedgerCard` owns two `useQuery` calls | Hooks remain structurally valid, requests start concurrently after the card mounts, and one failed count becomes `—` without affecting its sibling or card | Build count hooks in the parent map | Hook count would change with the ledger list and violate the Rules of Hooks |
| Keep the ID copy control outside the card's select button | `HashChip` is itself a button | Nest `HashChip` in a full-card button | Nested interactive controls are invalid HTML and ambiguous for keyboard users |
| Let Ledgers own its `PageHeader` while Shell owns the other route headers | The create drawer state and trigger ref remain local, and `+ New ledger` occupies the specified action slot | Add action state or callbacks to global AppState | The drawer is page-local UI, not global application state |
| Use named CSS grid breakpoints | The ticket requires exact 900 px and 1440 px viewport transitions | Standard Tailwind `md`/`xl` columns | Their breakpoint values do not match the acceptance contract |
| Keep zero new test dependencies | GRF-230 owns the DOM integration harness and the ticket forbids dependencies | Add jsdom/Testing Library for this page only | It would pre-empt a later ticket and violate dependency scope |

**Deviations from the ticket**

None. The dependency-free Node test suite verifies each render branch and pure boundary behavior but cannot synthesize browser focus/click transitions; those criteria were additionally exercised manually in the integrated browser. GRF-230 and GRF-232 remain responsible for automated DOM and end-to-end interaction coverage.

**Traps for future work**

- Per-card `api.changes` calls can currently return HTTP 500 for ledgers containing a DELETE Change because SQLite scans nullable `desired` into a non-nullable value. Count isolation correctly renders `—`; the owner-approved GRF-206 prerequisite fix will restore the count.
- Keep pure exports out of React component modules. Exporting the validation/count helpers from `ledgers-page.tsx` caused Vite to invalidate Fast Refresh until they moved to `ledgers-page.logic.ts`.
- The integrated browser did not close the native `<dialog>` through its default cancel path. The shared Drawer therefore handles `Escape` on keydown, updates controlled state, and lets its effect call `close()`; removing that handler breaks the documented keyboard path in this environment.
- The provider and page intentionally issue separate ledger-list reads: the provider owns navigation/switcher state, while the page owns loading, retry, and reconnect rendering. Do not couple page rendering to provider internals.

**Tests added**

- `studio/src/features/ledgers/ledgers-page.test.tsx` — three-card loading geometry, populated cards/counts/active semantics, empty action, HTTP error Retry surface, field-local conflict message, isolated count failure, trim/length validation, and exact READY filtering.

**Docs updated**

- `docs/ai/design-system.md` §5.1 — marked implemented and recorded the shipped interactions and breakpoints.
- `docs/ai/tickets/INDEX.md` — GRF-205 marked Done.
- `docs/ai/phases/phase-1.md` — ticket table and this completion record.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference       (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant  (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                    (cached)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  8 passed (8)
Tests       23 passed (23)
✓ 1868 modules transformed.
✓ built in 1.03s

$ docker build -t gyrifi:local .
[+] Building 32.1s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

Manual browser verification used a live Runtime and screenshots at 1440, 1180, 900, and 480 CSS-pixel widths. Computed grids were 3, 2, 2, and 1 columns respectively. The browser also verified: name focus on open; Escape close and focus restoration to `New ledger`; exact field-local `409` message; successful description persistence, drawer close, active-ledger switch, list/count refetch, and header-trigger focus; three-second `role="status"` removal; and card-button selection without route navigation.

**Follow-ups discovered**

- A combined `GET /api/v1/ledgers?include=counts` contract may be warranted if ledger cardinality makes the current two requests per visible card uncomfortable; do not add it without a ticket.
- GRF-230 should directly exercise Drawer focus transitions and the three-second confirmation under fake timers once its DOM harness exists. GRF-232 should retain the four-width screenshot path.

### GRF-206 — Changes page redesign

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — interaction automation remains with GRF-230/232 |

**What was built**

The Changes surface is now a durable inbox workspace with live status counts, client-side filters, a keyboard-navigable selectable table, full Change inspection, and explicit loading/empty/error/stale/populated states. READY selection starts an ordered Proposal flow, while a separate drawer accepts PUT or DELETE Changes with field-local JSON and idempotency errors. The prerequisite SQLite NULL-scan defect was fixed so one DELETE Change no longer makes the entire inbox unreadable.

**Files added**

- `studio/src/features/changes/changes-page.logic.ts` — pure filtering, counts, submission preparation, ordering, and idempotency-key helpers.
- `studio/src/features/changes/changes-page.logic.test.ts` — boundary and payload regressions.
- `studio/src/features/changes/changes-page.test.tsx` — rendered page-state and contract coverage.
- `studio/src/features/changes/selection-action-bar.tsx` — domain-owned sticky selected-count actions.
- `studio/src/features/shared/time.ts` — dependency-free relative age formatting.
- `studio/src/features/shared/time.test.ts` — relative-time boundary coverage.

**Files changed**

- `runtime/internal/repository/sqlite.go` — scans nullable Change desired bytes before assigning `json.RawMessage`.
- `runtime/internal/repository/sqlite_test.go` — protects DELETE list reads and omitted desired serialization.
- `studio/src/features/changes/changes-page.tsx` — rebuilt the complete inbox, detail, submission, and ordered Proposal flows.
- `studio/src/ui/patterns/data-table.tsx` — added row eligibility/reasons, ordered selection callbacks, keyboard row navigation, and accepted-row highlighting.
- `studio/src/app/providers.tsx` — added the shared ledger-switcher open request contract.
- `studio/src/features/shell/ledger-switcher.tsx` — opens and focuses in response to application requests.
- `studio/src/app/shell.tsx` — lets Changes own its header so the page action is rendered once.
- `docs/ai/design-system.md`, `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/tickets/INDEX.md`, `docs/ai/phases/phase-1.md` — current product, implementation, test, and completion state.

**Files removed**

None.

**Contracts introduced or changed**

```ts
interface AppState {
	openLedgerSwitcher: () => void;
	ledgerSwitcherRequest: number;
}

type DataTableProps<T> = {
	isRowSelectable?: (row: T) => boolean;
	getSelectionDisabledReason?: (row: T) => string;
	highlightedId?: string;
};

function formatAge(iso: string, now?: number): string;
function validateDesiredJson(value: string): string | undefined;
function prepareChangeSubmission(unit: string, action: Change["action"], desired: string, idempotencyKey: string): { input?: ChangeSubmission; jsonError?: string };
function moveOrdered(ids: string[], index: number, direction: -1 | 1): string[];
```

No HTTP or schema contract changed. `GET /api/v1/ledgers/{ledgerID}/changes` now fulfills its existing contract for rows whose nullable `desired` column is NULL; `json:"desired,omitempty"` continues to omit that field for DELETE Changes.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Fix the DELETE scan in GRF-206 | The inbox cannot meet its acceptance criteria if any valid DELETE row makes `ListChanges` return 500; owner approved folding in the prerequisite | Hide or filter DELETE rows in Studio | It would make the durable audit trail lie by omission |
| Keep selected IDs as an ordered array | Proposal identity is order-sensitive, and ordering must survive selection and reordering | Use a `Set` | Sets make the semantically meaningful order implicit and fragile |
| Start Proposal creation only from selected READY rows | The server remains authoritative while the UI provides the minimum documented eligibility hint | Reproduce claim and gate rules in the browser | Client governance rules drift; conflicts must come verbatim from the server |
| Add a provider request token for the topbar switcher | A no-ledger page action must open the already-owned switcher without moving its state into the page | Duplicate a ledger chooser inside Changes | Two selection surfaces would diverge and complicate focus handling |
| Keep the existing dependency-free test stack | GRF-230 owns the DOM integration harness and this ticket authorizes no dependency | Add jsdom/Testing Library for one page | It would pre-empt GRF-230 and violate ticket dependency scope |

**Deviations from the ticket**

The implementation meets every product acceptance criterion. The test-plan interactions are not synthesized through a DOM test harness: production-used submission preparation and the selected-count component are covered directly, while Space selection, Tab access to the action bar, inline parser errors, no-request invalid submission, and the real DELETE request body were verified in the integrated browser. GRF-230/232 retain automated rendered interaction and end-to-end coverage.

The design sketch showed a second `Build proposal →` header action and an idempotency key in detail. The ticket's selection-first acceptance contract is used instead, and the existing Change response does not expose an idempotency key; both design-level deviations are now explicit in design-system §5.2.

**Traps for future work**

- SQL NULL cannot scan directly into `json.RawMessage`; scan nullable blobs through `[]byte` and assign after `Scan`.
- The shared Shell must not also render a header for pages that own header actions, or the title and description appear twice.
- Keep `SelectionActionBar` and pure helpers outside the page component module so Fast Refresh sees only component exports and Vitest can cover decision boundaries without a browser DOM.
- A successful mutation calls `refetch()` without awaiting it; the accepted row arrives through authoritative REST and is highlighted by returned Change ID for three seconds.

**Tests added**

- `runtime/internal/repository/sqlite_test.go` — a persisted DELETE followed by `ListChanges` remains readable and serializes without `desired`.
- `studio/src/features/changes/changes-page.logic.test.ts` — exact status filtering/counts, invalid JSON blocking, PUT parsing, DELETE desired omission, ordered movement, and idempotency-key uniqueness.
- `studio/src/features/changes/changes-page.test.tsx` — rows and eligibility, five-state branches, no-ledger switcher action, stale visible content, selected-count bar, and field-local conflict text.
- `studio/src/features/shared/time.test.ts` — now/minute/hour/day boundaries plus future and malformed timestamps.

**Docs updated**

- `docs/ai/design-system.md` §5.2 — marked implemented and recorded design-sketch deviations.
- `docs/ai/product.md` §6 — Proposal creation now begins in Changes.
- `docs/ai/repo-structure.md` §3 — current feature/helper/test layout.
- `docs/ai/tech-spec.md` §5, §11–§12 — DELETE scan behavior, AppState switcher request, and tests.
- `docs/ai/tickets/INDEX.md` — GRF-206 marked Done.
- `docs/ai/phases/phase-1.md` — ticket table and this completion record.

**Verification**

```
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine             (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http    (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger              (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository          1.712s
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant      (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                        2.418s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  11 passed (11)
Tests       42 passed (42)
✓ 1873 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.29 kB
dist/assets/index-8-BVbHoB.css   41.27 kB │ gzip:  8.37 kB
dist/assets/index-CRif-rAB.js   295.51 kB │ gzip: 91.72 kB
✓ built in 1.02s

$ docker build -t gyrifi:local .
[+] Building 32.3s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /out/gyrifi ./cmd/gyrifi
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

Manual browser verification used the current Runtime with live PUT and DELETE rows. It confirmed one non-duplicated page header, three live stats, DELETE list rendering, Space row selection, `1 selected`, keyboard Tab reachability through `Clear` to `Create proposal`, invalid JSON's native parser message with zero POST requests, and a successful DELETE POST whose body contained `unit`, `action`, and `idempotencyKey` but no `desired`.

**Follow-ups discovered**

- GRF-214 must replace the visible client-side fetched-page filter limitation with bounded server pagination/filtering.
- GRF-230 should synthesize selection, blur/submit validation, drawer focus, timers, and switcher-open requests under jsdom; GRF-232 should retain the full keyboard flow in a real browser.

### GRF-207 — Proposals review workspace

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — current primary orange supersedes interim jade wording; gate copy remains server-authored |

**What was built**

Proposals is now a linkable two-pane review workspace. The left review queue shows status, title, Change count, and relative age with arrow-key navigation. The selected Proposal opens a detailed governance path containing identity, a four-step progress rail, ordered Changes with the shared drawer, persisted user criteria and complete evidence, editable hash-bound approval, and confirmed Release. A separate drawer creates Proposals from explicitly ordered READY Changes.

Every action uses authoritative Runtime state. The Studio projects `approvalAction` and `releaseAction` without recreating their rules, treats non-current checks and approvals as stale, surfaces server conflict text, and distinguishes an HTTP 503 recovery-required response from a Runtime transport outage.

**Files added**

- `studio/src/app/router.test.ts` — structured Proposal detail route coverage.
- `studio/src/features/changes/change-detail-drawer.tsx` — shared Change inspection reused by Changes and Proposals.
- `studio/src/features/proposals/{proposal-detail,evidence-panel,approval-panel,release-panel,progress-rail,create-proposal-drawer}.tsx` — decomposed review and creation surfaces.
- `studio/src/features/proposals/{proposal-view,gates,criteria-presets}.ts` — defensive evidence/progress projections, verbatim server gates, ordering, and presets.
- `studio/src/features/proposals/{proposals-page,proposal-detail,release-panel,create-proposal-drawer,gates}.test.ts(x)` — focused state and contract coverage.

**Files changed**

- `studio/src/api/{types,client,client.test}.ts` — complete evaluation result and editable approval actor contracts.
- `studio/src/app/{router,shell,reachability}.tsx` — structured detail routes, page-owned header, and Proposal event invalidation.
- `studio/src/features/proposals/proposals-page.tsx` — replaced flat cards and action buttons with the review queue/workspace.
- `studio/src/features/changes/changes-page.tsx` and `studio/src/ui/patterns/data-table.tsx` — extracted reusable Change detail and permitted Proposal selection behavior.
- `studio/src/features/shell/nav.tsx` and `studio/src/styles.css` — detail-route navigation and responsive two-pane layout.
- Reference docs, ticket status, and this phase log — current product and implementation state.

**Contracts introduced or changed**

```ts
type Route = { area: Area; id?: string };

function approvalGate(gates: ProposalGates): ActionGate;
function releaseGate(gates: ProposalGates): ActionGate;

interface EvaluationResponse {
	passed: boolean;
	summary: string;
	previewFidelity: string;
	findings?: Finding[];
	model?: string;
	evidence?: unknown;
}
```

`api.approve(ledgerId, proposalId, actor)` now sends the editable actor. No Runtime endpoint or persistence schema changed.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Project `approvalAction` and `releaseAction` by identity | GRF-211 deliberately centralizes governance and reason text in the Runtime | Recompute eligibility from status/checks/HEAD in React | Duplicate policy can drift or race with the release request |
| Keep REST authoritative and SSE advisory | Events signal invalidation but do not carry complete review state | Mutate local Proposal state from event payloads | Missed/reordered events could create false governance displays |
| Store criteria per Proposal and the actor globally | Criteria belongs to a hash review; actor represents the last operator identity | Hardcode both values | It prevents meaningful checks and silently attributes approvals to the wrong actor |
| Render HTTP 503 as durable recovery guidance | A responding Runtime may have persisted a recovery-required Intent | Treat 503 as offline or a transient toast | It misdiagnoses reachability and hides required operator work |
| Use native `details`/`summary` sections | Native semantics provide pointer plus Enter/Space expansion | Custom disclosure key handlers | They duplicate browser behavior and increase accessibility risk |

**Deviations from the ticket**

The current mockup-led orange primary token replaces the ticket's older jade wording. Disabled action text is not hardcoded to the ticket examples: the exact GRF-211 server reason is shown, preserving one policy authority. The browser does not compare base Release IDs itself; a moved-HEAD condition is represented by `releaseAction` and any racing `409` remains visible. Interaction automation remains scoped primarily to GRF-230/232; production helpers and rendered states are covered by the dependency-free Vitest harness and the complete path was exercised in the integrated browser.

**Traps for future work**

- Node can expose a partial `localStorage` object; guard `getItem` and `setItem` method availability in dependency-free SSR tests.
- Qdrant logical unit IDs must match point IDs. A deliberately mismatched unit produced a persisted recovery-required Intent and verified the 503 recovery banner; a numeric unit matching the desired point completed successfully.
- A successful Release is synchronous: the Runtime verifies the target and atomically records the Release, advances HEAD, marks Changes/Proposal RELEASED, and finalizes the Intent before returning 201.
- The dev image currently installs `air@latest`; Air v1.67.4 requires Go 1.26 while the image is Go 1.24. Production Compose remains valid and was used for live verification.

**Tests added**

- Route parsing for `#proposals/{id}` and fallback behavior.
- Review queue loading/empty/error/stale/populated rendering and arrow navigation.
- Defensive evidence parsing, progress states, order movement, exact server-gate projection, stale evidence, and stale approval.
- Ordered creation, server error display, confirmation requirement, HTTP-503 recovery banner, and transport-error distinction.
- API approval payload regression for the user-selected actor.

**Docs updated**

- `docs/ai/design-system.md` §5.3 — implemented structure and server-authored gate behavior.
- `docs/ai/product.md` §§6–7 — complete review workspace and removal of the hardcoded-criteria gap.
- `docs/ai/repo-structure.md` — split Proposal feature layout.
- `docs/ai/tech-spec.md` §§11–12 — detail hash route, evaluation/actor contracts, gate projection, and tests.
- `docs/ai/tickets/GRF-207-proposals-workspace.md` and `docs/ai/tickets/INDEX.md` — acceptance and status complete.

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine             (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http    (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger              (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository          (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant      (cached)
ok      github.com/gyrifi/gyrif-context-ledger/runtime/tests                        (cached)

$ cd studio && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  17 passed (17)
Tests       64 passed (64)
✓ 1865 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.29 kB
dist/assets/index-DD8_frDg.css   42.99 kB │ gzip:  8.69 kB
dist/assets/index-B510EoVd.js   295.97 kB │ gzip: 89.68 kB
✓ built in 1.00s

$ docker build -t gyrifi:local .
[+] Building 2.7s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /out/gyrifi ./cmd/gyrifi
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent
```

Manual browser verification against the production image and live Qdrant confirmed ledger selection, ordered Proposal creation, linkable detail reload, criteria/actor persistence, deterministic FAST evidence, exact server gate reasons, approval metadata, confirmation focus and consequences, a genuine HTTP-503 recovery-required banner, and a second complete Change → Proposal → Evaluation → Approval → verified Release path whose final state was `RELEASED` with HEAD advanced.

**Follow-ups discovered**

- GRF-213 must make the recovery banner's Releases destination actionable by listing and resolving persisted Intents.
- GRF-213 landed and GRF-208 now provides that operator surface inside Releases.
- GRF-230 should automate the loading skeleton, local-storage persistence, native disclosures, and confirmation focus lifecycle; GRF-232 should retain the live happy path plus recoverable 503 path.
- Pin the dev-only Air version or upgrade its Go builder in GRF-227 maintenance; `air@latest` no longer builds on Go 1.24.

### GRF-208 — Releases timeline and rollback flow

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — current orange primary replaces stale jade HEAD wording; absent pre-state is distinguished from missing rollback material |

**What was built**

Releases is now an immutable-history workspace rather than a raw list. The shared Timeline marks HEAD, joins source Proposal titles and finalized Intent plans, exposes plan inspection, and places non-HEAD rollback behind explicit forward-history confirmation. Recovery-required Intents produce an amber operator surface with verification-only retry, structured mismatch display, and note-required abandonment. Loading, empty, error, stale, populated, mutation-error, and mutation-success states remain inline and REST-authoritative.

**Files added**

- `studio/src/features/releases/release-timeline.tsx` — Timeline projection, Release-to-Intent join, and exact rollback unit-count derivation.
- `studio/src/features/releases/plan-drawer.tsx` — operation fingerprints, target metrics, and before-image availability.
- `studio/src/features/releases/rollback-dialog.tsx` — confirmed forward-history mutation and verbatim in-dialog errors.
- `studio/src/features/releases/recovery-banner.tsx` — recovery count, Intent inspection, retry/mismatch, and explicit abandonment flows.
- `studio/src/features/releases/releases-page.test.tsx`, `plan-drawer.test.tsx`, `rollback-dialog.test.tsx`, `recovery-banner.test.tsx` — focused rendering and calculation coverage.

**Files changed**

- `studio/src/features/releases/releases-page.tsx` — full workspace query, all five states, timeline orchestration, success handoff, and event invalidation.
- `studio/src/ui/patterns/confirm-dialog.tsx` — optional pending, disabled, and reason props for an in-place mutation.
- `studio/src/app/shell.tsx` — removed the transitional shell-owned Releases heading so the feature owns one PageHeader like the other pages.
- `docs/ai/design-system.md`, `docs/ai/product.md`, `docs/ai/repo-structure.md`, `docs/ai/tech-spec.md`, `docs/ai/tickets/GRF-208-releases-timeline.md`, `docs/ai/tickets/INDEX.md` — current behavior, contracts, acceptance, tree, and status.

**Files removed**

None.

**Contracts introduced or changed**

```ts
function intentForRelease(release: Release, intents: ReleaseIntent[]): ReleaseIntent | undefined;
function rollbackAffectedUnitCount(releases: Release[], targetReleaseId: string, intents: ReleaseIntent[]): number | undefined;

function ReleaseTimeline(props: {
	releases: Release[];
	proposals: Proposal[];
	intents: ReleaseIntent[];
	onViewPlan: (release: Release, intent: ReleaseIntent) => void;
	onRollback: (release: Release, affectedUnitCount: number) => void;
}): ReactNode;

function PlanDrawer(props: {
	open: boolean;
	onClose: () => void;
	release?: Release;
	operations?: ReleaseIntentOperation[];
}): ReactNode;

function RollbackDialog(props: {
	open: boolean;
	onClose: () => void;
	ledgerId: string;
	release?: Release;
	affectedUnitCount: number;
	onCreated: (proposal: Proposal) => void;
}): ReactNode;

function RecoveryBanner(props: {
	ledgerId: string;
	intents: ReleaseIntent[];
	onUpdated: () => void;
	onResolved?: (message: string) => void;
}): ReactNode;
```

`ConfirmDialog` gained optional `confirmLoading`, `confirmDisabled`, and `confirmTitle` props. Existing Release confirmation callers remain source-compatible.

**Final rollback dialog copy**

1. “This creates a **new proposal**; it does not rewind history.”
2. “**{n} units** will be restored to their state at this release.”
3. “The proposal must be evaluated, approved, and released like any other.”
4. “**HEAD will move forward** to a new release after verification.”

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Load Releases, Proposals, and all Intents as one query | Timeline titles, plan counts, rollback impact, and recovery must represent one refetched snapshot | Independent page queries | Partial resolution would show contradictory history and recovery data |
| Count unique units across every Release newer than the target | This matches the Runtime's newest-to-oldest reduction by unit | Sum operation counts | Repeatedly touched units would be over-counted |
| Disable rollback when any newer Release plan is unavailable | The UI must never guess a destructive action's impact | Display zero or a partial count | Either value would misrepresent the operation |
| Warn only when `beforeExists && !hasBeforeImage` | A unit absent before release needs no object; rollback correctly restores absence with DELETE | Treat every `hasBeforeImage: false` as data loss | It would falsely flag complete rollback plans for newly created units |
| Keep rollback API errors inside the open dialog | Operators retain context and server-authored remediation text | Close and toast | It hides whether any Proposal was created and loses the selected target |
| Treat SSE as a refetch hint | REST remains authoritative for mutable recovery state | Patch Intents from events | Events are lossy and do not carry Plans or mismatch data |

**Deviations from the ticket**

- The ticket says the HEAD node is jade, but the current design system normatively defines the primary node as orange. The implementation follows the current token contract and pairs it with a success-tone HEAD badge.
- `hasBeforeImage: false` is not automatically an amber failure. When `beforeExists` is false, no object is required and rollback restores absence with DELETE; only a missing required before-image displays “No rollback material for this unit.”
- No acceptance behavior was omitted.

**Traps for future work**

- Match a Release to its finalized Intent by Proposal ID. Abandoned attempts for the same Proposal must never supply the displayed Release plan.
- Keep Releases newest-first. `rollbackAffectedUnitCount` relies on every entry before the selected target being newer.
- Recovery mutations use one hook per operation type; retain the active Intent ID so an error or mismatch is rendered only on the row that initiated it.
- Native `<dialog>` owns the focus trap. Do not add destructive autofocus; keep errors in the dialog and close only after a returned Proposal.
- A false `hasBeforeImage` can mean either missing material or a correctly absent pre-state. Always inspect `beforeExists`.

**Tests added**

- Page tests cover required heading/copy, newest-first rendering, HEAD, source title, View/Rollback actions, loading, empty, HTTP error, stale data, and direct Proposal review handoff.
- Plan drawer tests cover operation identity/actions, both fingerprints, target metric, missing required before-image warning, and absent-pre-state explanation.
- Rollback dialog tests lock all four forward-history statements, verbatim 409/500 messages, destructive non-autofocus, unique-unit reduction, and missing-plan refusal.
- Recovery banner tests cover absence, plural count, Intent identity/status/plan projection, and both operator actions.

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race -count=1 && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine 1.578s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference 2.018s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http 2.571s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger 3.947s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository 3.210s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 3.521s
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 5.120s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files 21 passed (21)
Tests 74 passed (74)
✓ 1867 modules transformed.
✓ built in 799ms

$ docker build -t gyrifi:local .
[+] Building 32.9s (31/31) FINISHED
=> naming to docker.io/library/gyrifi:local

$ diff <ticket files> <INDEX status rows>
tickets consistent

$ git diff --check
(no output)
```

Manual browser inspection confirmed one feature-owned PageHeader, the Ledger-selection empty state, responsive shell placement, and no duplicate Releases heading. Populated behavior was additionally verified by focused component rendering; live recovery automation remains in GRF-232.

**Follow-ups discovered**

- GRF-230 should add interaction-level native-dialog focus, mutation-success, and recovery-action tests in a browser-like environment.
- GRF-232 should exercise a live failed Release through retry/mismatch and abandonment, plus rollback Proposal review handoff.

### GRF-240 — Mockup-led Studio product system

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | No |

**What was built**

The umbrella product system is now closed over the completed GRF-201…209 implementation rather than remaining an aspirational mockup contract. The design-system status, child work orders, and Phase 1 bookkeeping now describe the shipped light/orange, Tailwind-based Studio consistently. Shipped-image qualification exercises populated Changes and server-disabled Proposal gates at 1440, 1180, 900, and 480 px, preserving exact Runtime reasons and rejecting horizontal document overflow before the journey evaluates, approves, releases, rolls back, restarts, and deep-links.

Before this closure, the approved visual system was implemented but still documented as in progress, two historical work orders prescribed jade/BEM details, GRF-208 was stale in the phase table, and responsive evidence covered journeys without systematically traversing every canonical width. After it, each child links to this umbrella, no child prescribes the retired implementation, and one real-image test locks the representative populated and exceptional states at all four widths.

**Files added**

None.

**Files changed**

- `e2e/tests/governance.spec.ts` — added canonical-width qualification to the complete shipped-image governance journey.
- `docs/ai/design-system.md` — made the implemented visual contract current and recorded responsive browser evidence.
- `docs/ai/tech-spec.md` — documented the canonical viewport coverage in the E2E contract.
- `docs/ai/tickets/GRF-202-ui-library.md` … `GRF-209-studio-resilience.md` — linked every child to the umbrella and replaced remaining retired jade/BEM prescriptions.
- `docs/ai/tickets/GRF-232-e2e-suite.md` — recorded the additional responsive qualification owned by this umbrella.
- `docs/ai/tickets/GRF-240-mockup-led-studio-product-system.md` — marked every acceptance criterion complete.
- `docs/ai/tickets/INDEX.md` — marked GRF-240 done.
- `docs/ai/phases/phase-1.md` — closed the phase table/exit criteria and added this completion record.

**Files removed**

None.

**Contracts introduced or changed**

No product API, exported Studio type, database, or configuration contract changed. The browser suite gained this internal qualification contract:

```ts
async function qualifyCanonicalViewports(
	page: Page,
	proposal: Proposal,
	visibleUnit: string,
): Promise<void>;
```

For each width in `[1440, 1180, 900, 480]`, it requires one populated Change, the disabled `Approve` and `Release to Qdrant` controls, the exact Runtime-authored reasons, and `document.documentElement.scrollWidth <= width`.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Qualify widths inside the real governance journey | The assertions see real REST state, server gates, shipped assets, and container geometry | Add jsdom-only responsive tests | jsdom does not lay out the document or qualify the production image |
| Check a populated data surface and a disabled exceptional surface at every width | These are the densest information and governance states required by the ticket | Check only page headings or empty states | Presence alone would miss hidden reasons, clipped data, and overflow |
| Assert overflow numerically rather than accept screenshots as the gate | The invariant is deterministic and repeatable across all runs | Rely on manual screenshot review | Pixel review is subjective and does not block regressions automatically |
| Re-baseline child work orders narrowly | Historical acceptance intent remains useful while GRF-240 must be the current visual authority | Rewrite every child ticket from scratch | Broad historical edits would obscure what each ticket originally delivered |

**Deviations from the ticket**

None. Every acceptance criterion was met without a new dependency or a client-side governance decision.

**Traps for future work**

- Responsive qualification must occur before the first Proposal is evaluated; that state exposes both approval and release gates with distinct exact Runtime reasons.
- Keep viewport height fixed while varying width so failures isolate horizontal adaptation rather than vertical content availability.
- A document-width assertion complements visibility checks: individual controls can remain visible while another surface silently creates page-level horizontal overflow.
- On this macOS host, one coverage run reported all 144 tests and the complete threshold report, then hit a transient `kill EPERM` while Vitest terminated a worker. An immediate unchanged rerun exited cleanly; do not weaken coverage or assertions in response to that host-level teardown error.

**Tests added**

- `e2e/tests/governance.spec.ts` — the complete governance test now protects populated Changes, disabled Proposal actions, verbatim gate reasons, and no document overflow at 1440/1180/900/480 px. `test:repeat` runs this path three consecutive times.

**Docs updated**

- `docs/ai/design-system.md` §0 — promoted the mockup-led contract from in-progress target to browser-qualified implementation.
- `docs/ai/tech-spec.md` §13 — expanded the shipped-image suite contract with canonical viewport states.
- `docs/ai/tickets/GRF-202-ui-library.md` … `GRF-209-studio-resilience.md` — established GRF-240 as the visual authority and retired stale implementation wording.
- `docs/ai/tickets/GRF-232-e2e-suite.md` — documented responsive coverage layered onto its journey.
- `docs/ai/tickets/GRF-240-mockup-led-studio-product-system.md`, `docs/ai/tickets/INDEX.md`, and `docs/ai/phases/phase-1.md` — completed ticket and phase bookkeeping.

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
?  github.com/gyrifi/gyrif-context-ledger/runtime/cmd/gyrifi [no test files]
?  github.com/gyrifi/gyrif-context-ledger/runtime/internal/bootstrap [no test files]
?  github.com/gyrifi/gyrif-context-ledger/runtime/internal/config [no test files]
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests (cached)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test
Scope: all 2 workspace projects
Already up to date
Test Files  47 passed (47)
Tests       144 passed (144)
Duration    5.29s

$ pnpm coverage
Test Files  47 passed (47)
Tests       144 passed (144)
All files   86.2% statements | 85.83% branches | 71.14% functions | 86.2% lines

$ pnpm build
✓ 1867 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.29 kB
dist/assets/index-CXd8xUMU.css   42.89 kB │ gzip:  8.70 kB
dist/assets/index-hiFzMPQp.js   306.22 kB │ gzip: 91.94 kB
✓ built in 839ms

$ cd ../e2e && pnpm install --ignore-workspace --frozen-lockfile && pnpm --ignore-workspace test:repeat
Already up to date
Running 6 tests using 1 worker
✓ fresh shipped image shows the empty first-run path × 3
✓ shipped image governs, rolls back, restarts, and deep-links × 3
6 passed (46.9s)

$ cd .. && docker build -t gyrifi:local .
[+] Building 2.0s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /out/gyrifi ./cmd/gyrifi
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent

$ git diff --check
(no output)
```

**Follow-ups discovered**

None. CI enforcement remains GRF-233 and was not duplicated here.
