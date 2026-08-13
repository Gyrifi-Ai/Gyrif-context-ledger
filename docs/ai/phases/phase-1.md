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

_No entries yet. Use the template in [README.md](README.md) and append newest last._
