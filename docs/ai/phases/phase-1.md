# Phase 1 — Studio experience

**Goal:** turn the functional-but-unfinished Studio into an interface that makes the governance model obvious. The runtime already enforces the rules; Phase 1 makes them visible, explicable, and hard to get wrong.

**Status:** Not started

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-201](../tickets/GRF-201-design-tokens.md) | Design token foundation and stylesheet split | M | — | Not started |
| [GRF-202](../tickets/GRF-202-ui-library.md) | UI primitive and pattern library | L | GRF-201 | Not started |
| [GRF-203](../tickets/GRF-203-application-shell.md) | Application shell, navigation, real runtime status | M | GRF-202 | Not started |
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
