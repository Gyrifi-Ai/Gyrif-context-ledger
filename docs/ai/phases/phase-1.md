# Phase 1 — Studio experience

**Goal:** turn the functional-but-unfinished Studio into a mockup-led, light SaaS interface that makes the governance model obvious. The runtime already enforces the rules; Phase 1 makes them visible, explicable, and hard to get wrong.

**Status:** Not started

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-240](../tickets/GRF-240-mockup-led-studio-product-system.md) | Mockup-led Studio product system | XL | — | In progress |
| [GRF-201](../tickets/GRF-201-design-tokens.md) | Mockup-led design token foundation | M | — | Done |
| [GRF-202](../tickets/GRF-202-ui-library.md) | UI primitive and pattern library | L | GRF-201 | Done |
| [GRF-203](../tickets/GRF-203-application-shell.md) | Application shell, navigation, real runtime status | M | GRF-202 | Done |
| [GRF-204](../tickets/GRF-204-async-data-layer.md) | Async data layer | M | — | Not started |
| [GRF-205](../tickets/GRF-205-ledgers-page.md) | Ledgers page redesign | M | GRF-203, GRF-204 | Not started |
| [GRF-206](../tickets/GRF-206-changes-page.md) | Changes inbox redesign | L | GRF-203, GRF-204 | Not started |
| [GRF-207](../tickets/GRF-207-proposals-workspace.md) | Proposals review workspace | XL | GRF-203, GRF-204, GRF-211 | Not started |
| [GRF-208](../tickets/GRF-208-releases-timeline.md) | Releases timeline and rollback flow | L | GRF-203, GRF-204, GRF-213 | Not started |
| [GRF-209](../tickets/GRF-209-studio-resilience.md) | Studio resilience: error boundary, offline state, stream reconnection | M | GRF-202, GRF-204 | Not started |

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
