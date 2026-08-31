# Phase 1 — Studio experience

**Goal:** turn the functional-but-unfinished Studio into a mockup-led, light SaaS interface that makes the governance model obvious. The runtime already enforces the rules; Phase 1 makes them visible, explicable, and hard to get wrong.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-240](../tickets/GRF-240-mockup-led-studio-product-system.md) | Mockup-led Studio product system | XL | — | In progress |
| [GRF-201](../tickets/GRF-201-design-tokens.md) | Mockup-led design token foundation | M | — | Done |
| [GRF-202](../tickets/GRF-202-ui-library.md) | UI primitive and pattern library | L | GRF-201 | Done |
| [GRF-203](../tickets/GRF-203-application-shell.md) | Application shell, navigation, real runtime status | M | GRF-202 | Done |
| [GRF-204](../tickets/GRF-204-async-data-layer.md) | Async data layer | M | — | Done |
| [GRF-205](../tickets/GRF-205-ledgers-page.md) | Ledgers page redesign | M | GRF-203, GRF-204 | Not started |
| [GRF-206](../tickets/GRF-206-changes-page.md) | Changes inbox redesign | L | GRF-203, GRF-204 | Not started |
| [GRF-207](../tickets/GRF-207-proposals-workspace.md) | Proposals review workspace | XL | GRF-203, GRF-204, GRF-211 | Not started |
| [GRF-208](../tickets/GRF-208-releases-timeline.md) | Releases timeline and rollback flow | L | GRF-203, GRF-204, GRF-213 | Not started |
| [GRF-209](../tickets/GRF-209-studio-resilience.md) | Studio resilience: error boundary, offline state, stream reconnection | M | GRF-202, GRF-204 | Done |

## Phase-level notes

- **Two tickets reach into Phase 2.** GRF-207 needs GRF-211 (proposal detail and evidence reads) and GRF-208 needs GRF-213 (release intent inspection). Do not fake that data client-side to unblock the UI — a governance surface that displays locally-remembered state is worse than one that displays nothing.
- **Do GRF-209 early**, right after the component library. Every screen built before it inherits a shell that fails with a white page, and retrofitting error boundaries is harder than building on top of them.
- The full [design-system.md](../design-system.md) is normative for this phase. Deviations from it are deviations from the ticket and must be logged.
- No new frontend dependencies are permitted in this phase. React and react-dom remain the only runtime dependencies.

## Exit criteria

- [ ] All nine tickets complete.
- [ ] Every page implements all five interaction states from design-system §6.
- [ ] The full workflow — create ledger, review changes, propose, evaluate, approve, release, roll back — is completable in the browser without reading API documentation.
- [ ] Every disabled action states its reason on screen.
- [ ] Keyboard-only operation of the entire workflow is possible.
- [ ] No failure is silent: a render error, an unreachable runtime, and a dropped event stream each produce a visible, recoverable state.
- [ ] `pnpm typecheck && pnpm test && pnpm build` green.

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
$ cd studio && pnpm typecheck && pnpm test && pnpm build
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
