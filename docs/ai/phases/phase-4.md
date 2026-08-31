# Phase 4 — Qualification

**Goal:** replace assumption with evidence. The runtime's correctness is currently asserted by unit tests against fakes and by nothing else. This phase builds the machinery that proves the shipped artefact does what the documentation claims.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-233](../tickets/GRF-233-ci-pipeline.md) | CI pipeline | M | — | Not started |
| [GRF-230](../tickets/GRF-230-studio-tests.md) | Studio component and integration test suite | L | GRF-202 | Done |
| [GRF-231](../tickets/GRF-231-qdrant-qualification.md) | Qdrant integration qualification | L | — | Not started |
| [GRF-232](../tickets/GRF-232-e2e-suite.md) | Browser end-to-end qualification | L | GRF-205 … GRF-208 | Not started |

## Phase-level notes

- **GRF-233 is first, ahead of every ticket in every phase.** Each ticket's definition of done requires a green quality gate; without CI that is an honour system. It is the first ticket in the recommended order in [INDEX.md](../tickets/INDEX.md) for this reason.
- GRF-233 ships `integration` and `e2e` job stubs, disabled. GRF-231 and GRF-232 enable them and make them required checks. Enabling is part of those tickets, not a follow-up.
- GRF-231 may discover that the adapter's assumptions about Qdrant are wrong. That is the point. Any such discovery is a finding to record in `tech-spec.md` §9 and, if it is a defect, to fix inside the ticket.
- Inference (Gemma) is deliberately excluded from required CI paths. Model files are large and the product works without them. Any inference-dependent job is advisory and externally provisioned.
- New dependencies are permitted in this phase, confined to test tooling: Testing Library and jsdom (GRF-230), Playwright (GRF-232), coverage provider (GRF-230). Nothing enters the runtime dependency graph.

## What each ticket proves

| Ticket | Claim it makes verifiable |
|---|---|
| GRF-233 | Every commit satisfies the documented quality gate |
| GRF-230 | The UI enforces the governance gates it displays |
| GRF-231 | The adapter reads back exactly what it wrote to a real Qdrant |
| GRF-232 | The shipped image completes the full workflow and survives restart |

The four together mean the audit trail can be trusted end to end. Any one missing leaves a link unverified.

## Exit criteria

- [ ] All four tickets complete.
- [ ] CI runs the full gate plus integration and e2e as required checks.
- [ ] Frontend coverage thresholds met and enforced.
- [ ] The Qdrant adapter is verified against a pinned real Qdrant, including the cosine normalisation tolerance and partial-failure behaviour.
- [ ] The built Docker image is verified to complete a full release and rollback and to persist across a restart.
- [ ] Three consecutive clean runs of the e2e suite pass without flakiness.

## Completed entries

### GRF-230 — Studio component and integration test suite

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — local coverage enforcement landed, but CI invocation remains owned by GRF-233 |

**What was built**

Studio now has a browser-like jsdom test environment, Testing Library interaction helpers, a typed fetch-level Runtime mock, and co-located tests for every domain-free UI module. Integration tests exercise the providers, async lifecycle, application shell, and all four feature pages through their loading, empty, populated, error, and disabled states. V8 coverage is a direct-entry package command with enforced global thresholds; the completed suite has 47 files, 144 tests, 86.20% statement coverage, and 85.83% branch coverage.

The interaction suite exposed focus defects that server rendering could not reveal. The native confirmation dialog now contains keyboard focus and both confirmation and drawer dialogs explicitly handle Escape and restore focus to their opening trigger.

**Files added**

- `studio/src/test/setup.ts` — jest-dom registration, deterministic DOM shims, cleanup, API reset, and unexpected-console-error failure
- `studio/src/test/render.tsx` — provider-wrapped Testing Library render with a configured `userEvent`
- `studio/src/test/api-mock.ts` — typed fetch router covering the complete `Api` surface
- `studio/src/app/{providers,reachability-provider,use-async,use-ledger-events}.test.tsx` — provider, health, query, mutation, and event-invalidation integration coverage
- `studio/src/components/ui/components.test.tsx` — focused composition and interaction coverage for the shadcn component set
- `studio/src/features/shell/shell.test.tsx` — navigation, Ledger switching, identity, and connection-state behavior
- `studio/src/ui/{primitives,patterns,layout,feedback}/*.test.tsx` — co-located coverage for every domain-free component

**Files changed**

- `studio/package.json` and `pnpm-lock.yaml` — authorised jsdom, Testing Library, jest-dom, user-event, and matching V8 coverage tooling plus the direct-entry `coverage` script
- `studio/vite.config.ts` — jsdom setup, CSS processing, explicit globals policy, source selection, reporters, and 80/75 coverage thresholds
- `studio/src/api/client.ts` — exported the real `Api` type so test doubles fail compilation when its surface changes
- `studio/src/ui/patterns/confirm-dialog.tsx` — focus containment, Escape dismissal, trigger focus restoration, and single close notification
- `studio/src/ui/layout/drawer.tsx` — Escape dismissal, trigger focus restoration, and single close notification
- `studio/src/features/{ledgers,changes,proposals,releases}/**/*.test.tsx` — expanded rendered page-state, form, drawer, gate, rollback, and recovery interactions
- `studio/src/features/shared/status.test.ts` — compile-time-exhaustive domain-status tone expectations
- `studio/src/ui/feedback/async-boundary.test.tsx` — stale-content behavior asserted accessibly rather than through a CSS class
- `docs/ai/tech-spec.md`, `docs/ai/repo-structure.md`, `docs/ai/tickets/GRF-230-studio-tests.md`, `docs/ai/tickets/INDEX.md`, and this file — test contract, tree, acceptance, and completion records

**Files removed**

None.

**Contracts introduced or changed**

```ts
// studio/src/api/client.ts
export type Api = typeof api;

// studio/src/test/api-mock.ts
export type MockApi = { [K in keyof Api]: Mock<Api[K]> };
export const mockApi: MockApi;
export function resetApiMock(): void;

// studio/src/test/render.tsx
export function renderWithProviders(
	ui: ReactElement,
	options?: Omit<RenderOptions, "wrapper">,
): RenderResult & { user: ReturnType<typeof userEvent.setup> };
```

`pnpm coverage` is the stable CI entry point and executes `node node_modules/vitest/vitest.mjs run --coverage`. Its V8 configuration includes `src/**/*.{ts,tsx}`, excludes entry declarations, shared test infrastructure, and test modules, and rejects totals below 80% statements or 75% branches. No Runtime API, persistence, or governance contract changed.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Route real `fetch` calls into a complete mapped `MockApi` | Tests cover request serialization while TypeScript enforces every client method and signature | Mock or monkey-patch `api` methods inside feature modules | It bypasses the client boundary and drifts silently when method signatures change |
| Keep shared browser shims deterministic and reset them after every test | jsdom's dialog, EventSource, and Storage support does not provide the exact application lifecycle under test | Add a general test-utility or service-worker dependency | The ticket authorises a narrow dependency set and the app needs only small local substitutes |
| Co-locate behavior tests with subjects | Ownership and omissions remain visible when components are added | Central component-test directory | It separates coverage from the module and makes future additions easier to miss |
| Assert gates through role, disabled state, title, and visible server reason | Governance correctness is an accessible behavior, not a style | Assert utility classes or only test the pure projection helper | Class assertions are brittle and a pure helper cannot prove that the operator sees the Runtime decision |
| Add direct coverage execution rather than include coverage in `pnpm test` | Fast normal tests remain available while CI gets a stable enforcing command | Run every suite under coverage by default | It doubles instrumentation overhead for ordinary local iteration |
| Explicitly implement native-dialog keyboard/focus lifecycle | jsdom interaction tests exposed missing focus restoration and incomplete Escape behavior | Trust `<dialog>` behavior without component handling | Browser implementations and test environments differ, and callers need one reliable close notification |

**Deviations from the ticket**

The 80% statement and 75% branch thresholds are configured and enforced by `pnpm coverage`, but that command is not yet invoked by a CI workflow because GRF-233 has not landed and the repository has no workflow to edit. GRF-233 must call this command; duplicating or prematurely creating its workflow is outside GRF-230 scope. All other acceptance criteria were met.

**Traps for future work**

- A `Response` body is single-use. Fetch mocks must create a fresh response for every call rather than resolve repeated requests to one shared instance.
- jsdom may expose incomplete browser globals. Replace Storage with one deterministic implementation instead of assuming its methods are present, and keep dialog shims limited to missing native methods.
- `userEvent.type()` treats braces as keyboard descriptors; paste JSON payloads when testing editors that accept arbitrary JSON.
- React rerenders can call mocked hooks more than once. Fixture sequencing must model component identity rather than rely on a monotonically increasing global call number.
- Browser native `required` validation runs before application submit handlers. Use whitespace input to reach trim-based application validation.
- Native dialog `close` events must not invoke a caller callback a second time after the component already requested closure.

**Tests added**

- `studio/src/ui/**/*.test.tsx` — every domain-free component's variants, accessible states, actions, and applicable keyboard behavior
- `studio/src/components/ui/components.test.tsx` — shadcn component rendering and Radix interaction contracts
- `studio/src/app/use-async.test.tsx` — query loading/success/error/unavailable/refetch/abort and mutation pending/duplicate/offline/error/reset paths
- `studio/src/app/providers.test.tsx` and `reachability-provider.test.tsx` — Ledger persistence/refresh/switcher requests and Runtime HTTP reachability
- `studio/src/app/use-ledger-events.test.tsx` — invalidation registration and cleanup
- `studio/src/features/shell/shell.test.tsx` — gated navigation, keyboard switcher operation, HEAD errors, and Runtime/event-stream controls
- `studio/src/features/ledgers/ledgers-page.test.tsx` — five states, validation, trimmed create payload, switcher selection, and unreachable mutation gate
- `studio/src/features/changes/changes-page.test.tsx` — five states, filters, status rendering, row detail, selection, ordered Proposal flow, and Change form payloads
- `studio/src/features/proposals/{proposals-page,proposal-detail}.test.tsx` — five states and rendered server-authored gate reasons
- `studio/src/features/releases/releases-page.test.tsx` — five states, four rollback consequences, unavailable-plan refusal, and recovery-banner presence rules

**Docs updated**

- `docs/ai/tech-spec.md` §§11–13 — exported test contract, complete Studio test surface, and enforcing coverage command
- `docs/ai/repo-structure.md` §3–§4 — shared test files, co-location convention, and direct-entry coverage script
- `docs/ai/tickets/GRF-230-studio-tests.md` — acceptance results and explicit GRF-233 CI deferral
- `docs/ai/tickets/INDEX.md` — GRF-230 marked Done
- `docs/ai/phases/phase-4.md` — Phase 4 marked in progress and this completion record

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine 2.172s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 2.635s
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 4.386s

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm coverage && pnpm build
Scope: all 2 workspace projects
Lockfile passes supply-chain policies
Test Files  47 passed (47)
Tests       144 passed (144)
All files  | 86.20 % Stmts | 85.83 % Branch | 71.14 % Funcs | 86.20 % Lines
✓ 1867 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.29 kB
dist/assets/index-CXd8xUMU.css   42.89 kB │ gzip:  8.70 kB
dist/assets/index-hiFzMPQp.js   306.22 kB │ gzip: 91.94 kB
✓ built in 905ms

$ docker build -t gyrifi:local .
[+] Building 64.6s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /out/gyrifi ./cmd/gyrifi
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent

$ git diff --check
(no output)
```

**Follow-ups discovered**

- GRF-233 must invoke `pnpm coverage` in the Studio CI job so the thresholds become a required continuous check.
- GRF-232 remains responsible for native-dialog, focus, full governance workflow, recovery, restart, and persistence qualification in real Chromium against the shipped image; jsdom tests do not replace that browser/runtime boundary.
