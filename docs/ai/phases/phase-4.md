# Phase 4 — Qualification

**Goal:** replace assumption with evidence. The runtime's correctness is currently asserted by unit tests against fakes and by nothing else. This phase builds the machinery that proves the shipped artefact does what the documentation claims.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-233](../tickets/GRF-233-ci-pipeline.md) | CI pipeline | M | — | Done |
| [GRF-230](../tickets/GRF-230-studio-tests.md) | Studio component and integration test suite | L | GRF-202 | Done |
| [GRF-231](../tickets/GRF-231-qdrant-qualification.md) | Qdrant integration qualification | L | — | Done |
| [GRF-232](../tickets/GRF-232-e2e-suite.md) | Browser end-to-end qualification | L | GRF-205 … GRF-208 | Done |

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

- [x] All four tickets complete.
- [ ] CI runs the full gate plus integration and e2e as required checks.
- [ ] Frontend coverage thresholds met and enforced.
- [x] The Qdrant adapter is verified against a pinned real Qdrant, including the cosine normalisation tolerance and partial-failure behaviour.
- [ ] The built Docker image is verified to complete a full release and rollback and to persist across a restart.
- [ ] Three consecutive clean runs of the e2e suite pass without flakiness.

## Completed entries

### GRF-233 — CI pipeline

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | `b01c3cc`; branch push run [33430112043](https://github.com/Gyrifi-Ai/Gyrif-context-ledger/actions/runs/33430112043) succeeded; PR unavailable under Enterprise Managed User policy |
| Deviated from ticket | Yes — qualification used a real feature-branch push rather than a pull request; destructive gate, cache-timing, and cancellation experiments were not performed |

**What was built**

A single least-privilege GitHub Actions workflow now enforces the repository gate on every push and pull request. Runtime and Studio run independently with pinned toolchains and dependency caches; Image waits for both, builds through Buildx with traceable metadata, polls the real Runtime for health, validates its reported version, and uploads the image for downstream qualification. Commented integration and e2e extension points remain disabled and identify the tickets and repository settings needed to make them required checks.

**Files added**

- `.github/workflows/ci.yml` — Runtime, Studio, Image, disabled Qdrant integration, and disabled browser e2e jobs

**Files changed**

- `README.md` — CI status badge and enforced-gate documentation
- `AGENTS.md` — identifies the workflow-enforced gate and its intentionally disabled extensions
- `docs/ai/tech-spec.md` — CI contract, pinned environments, commands, dependency graph, image handoff, and extension ownership
- `docs/ai/repo-structure.md` — workflow ownership and path
- `docs/ai/tickets/GRF-233-ci-pipeline.md`, `docs/ai/tickets/INDEX.md`, and this file — verified acceptance results and completion records

**Files removed**

None.

**Contracts introduced or changed**

The stable workflow surface is `.github/workflows/ci.yml` with required jobs named `Runtime`, `Studio`, and `Image`. `Image` has `needs: [runtime, studio]`. It exports `gyrifi-ci-image`, containing `gyrifi-ci.tar.gz`, for one day. Runtime uses Go from `runtime/go.mod`; Studio uses Node 24 and pnpm 11.15.1. `pnpm coverage` is now continuously enforced.

Pushes trigger on every branch, not only the default branch. This is intentionally broader than the minimum contract and lets contributors validate real workflow execution even when organization policy prevents pull-request creation.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Use immutable SHAs for all actions | Mutable action tags are a supply-chain risk in the repository's quality authority | Pin release tags only | Tags can be moved after review |
| Keep Runtime and Studio parallel and gate Image on both | Shortens the common failure path and avoids expensive image work after a source failure | One serial quality-gate job | It increases total duration and obscures ownership |
| Use setup-action caches plus Buildx GHA layers | These caches are supported, scoped by dependency inputs, and need no custom persistence code | Hand-managed cache paths and keys | More maintenance with weaker tool integration |
| Poll Runtime and install no container healthcheck solely for CI | The existing system-status endpoint is the product contract and the wait remains bounded | Fixed sleep or Dockerfile-only CI healthcheck | Sleeps are flaky; adding shipping behavior only for CI is unnecessary scope |
| Export a compressed daemon image | It gives GRF-232 a direct handoff without publishing credentials | Publish to a registry | Publishing and credentials are explicitly out of scope |
| Leave both qualification stubs disabled | GRF-231 owns real Qdrant qualification; the user explicitly required GRF-233's e2e stub to remain disabled | Enable the already-existing e2e package now | It contradicts the selected ticket boundary |

**Deviations from the ticket**

- The workflow was green on real push run 33430112043, not a pull request. GitHub rejected PR creation because the authenticated account is an Enterprise Managed User that cannot access the pull-request operation. Branch-push validation exercises the same workflow jobs, but does not prove the `pull_request` event path; that trigger is present in the workflow.
- The test-plan experiments that intentionally introduce an unformatted file, vet error, Go/Vitest failure, TypeScript error, and broken Dockerfile were not run against the shared branch. The corresponding commands and failure-preserving shell settings are present, and the unmodified gate completed successfully.
- Cold-versus-warm cache timing and rapid-push concurrency cancellation were not deliberately exercised. Cache save/restore wiring and ref-scoped `cancel-in-progress: true` are statically present; no stronger empirical claim is made.
- The acceptance wording requested push to the default branch. The workflow listens to all push branches, a strict superset needed to qualify this branch under the account restriction.
- All implementation acceptance criteria were met. The e2e job remains a commented disabled extension point, per the user's explicit instruction to follow GRF-233 literally.

**Traps for future work**

- The account can push branches and run Actions but cannot create a pull request; do not misreport branch-push evidence as PR evidence.
- Keep Studio package scripts on direct Node entry points. The local workspace colon makes `node_modules/.bin` unavailable through PATH.
- `load: true` is required before smoke testing or saving a Buildx result from the runner daemon.
- Repository settings, not workflow YAML, make an enabled job a required check. GRF-231 and the eventual e2e activation must update both.
- `BUILD_DATE` is computed by the workflow because GitHub does not provide the required RFC 3339 value directly.

**Tests added**

No source test modules were added. The workflow itself executes formatting and module-drift checks, Go vet/race/build, Studio typecheck/test/coverage/build, an actual image build, a bounded HTTP startup probe, exact version comparison, container cleanup, image export, and artifact upload.

**Docs updated**

- `docs/ai/tech-spec.md` §13 — complete enforced workflow contract
- `docs/ai/repo-structure.md` — workflow location and ownership
- `README.md` — badge and contributor-facing enforcement note
- `AGENTS.md` §5 — CI enforcement and disabled-extension status
- `docs/ai/tickets/GRF-233-ci-pipeline.md` — acceptance results and deviations
- `docs/ai/tickets/INDEX.md` — GRF-233 marked Done
- `docs/ai/phases/phase-4.md` — this completion record

**Verification**

```text
$ GitHub Actions push run 33430112043 @ b01c3cc
Runtime: gofmt and go mod tidy clean; go vet passed; go test ./... -race passed; go build ./... passed
Studio: pnpm install --frozen-lockfile, typecheck, test, coverage, and build passed
Test Files  47 passed (47)
Tests       144 passed (144)
All files  | 91.67% Stmts | 85.94% Branch | 71.47% Funcs | 91.67% Lines
✓ built in 2.16s
Image: Buildx build and bounded HTTP smoke test passed (`true`)
Artifact gyrifi-ci-image uploaded: ID 9772390493, 59,215,088 bytes
Run conclusion: success

$ local quality gate
Runtime: gofmt, go vet, go test ./... -race, and go build ./... passed
Studio: frozen install, typecheck, 47 files / 144 tests, coverage, and production build passed
All files  | 86.20% Stmts | 85.83% Branch | 71.14% Funcs | 86.20% Lines
Docker: 31/31 build steps passed; embedded Runtime HTTP smoke test passed
Ticket INDEX consistency: tickets consistent
git diff --check: no output
```

**Follow-ups discovered**

- GRF-223 replaces the initial non-empty image-version smoke assertion with exact linker-injected version equality. This follow-up was completed in commit `b09a8d0`.
- GRF-231 must enable and qualify the Qdrant integration job and make it required.
- Browser e2e CI activation remains separate work; GRF-232 supplied the suite, but this ticket deliberately retained the disabled extension point as requested.

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

### GRF-232 — Browser end-to-end qualification

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Autonomous checkpoint; owner review pending |
| Deviated from ticket | Yes — CI enforcement remains owned by the not-yet-landed GRF-233 workflow |

**What was built**

An isolated Playwright package now builds the current repository Dockerfile, starts that image with pinned Qdrant, waits for a healthy Runtime, and drives Chromium through the product. Two expensive journeys cover an empty first run and the complete Ledger creation, API ingestion, browser Proposal review, deterministic evaluation, approval, Release, second Release, rollback-as-forward-Proposal, third Release, Qdrant verification, persistence, graceful SIGTERM, and SPA fallback path. Each test gets fresh named volumes, a unique collection, and its own Ledger; the suite passed three consecutive isolated repetitions in 45.9 seconds.

The browser-connected SIGTERM check exposed a production shutdown defect: an active SSE request kept `http.Server.Shutdown` waiting until its ten-second deadline and made the container exit 1. The server now derives every request context from the process signal context, so the stream exits before graceful shutdown waits and the container exits 0 with a non-growing WAL.

**Files added**

- `e2e/package.json` and `e2e/pnpm-lock.yaml` — isolated pinned Playwright package with direct-entry test, stability, and Chromium-install commands
- `e2e/playwright.config.ts` — Chromium-only, one-worker, bounded suite with retained failure traces/screenshots
- `e2e/compose.yaml` — current-image build, pinned Qdrant/initializer, loopback ports, named volumes, and configurable per-test collection
- `e2e/tests/global-setup.ts` and `global-teardown.ts` — image build, initial health gate, and deterministic cleanup
- `e2e/tests/harness.ts` — Compose lifecycle, status polling, per-test isolation, SIGTERM/WAL inspection, ingestion, and direct Qdrant reads
- `e2e/tests/governance.spec.ts` — empty-first-run and full governance/rollback/restart/deep-link journeys

**Files changed**

- `runtime/internal/bootstrap/bootstrap.go` — HTTP request contexts now inherit process cancellation so SSE cannot prevent graceful shutdown
- `.gitignore` — excludes Playwright reports and per-failure result artifacts
- `README.md` — local e2e prerequisites, isolated install, invocation, ports, artifacts, and three-run command
- `docs/ai/tech-spec.md` — Playwright version, request-lifecycle shutdown contract, test surface, and e2e quality command
- `docs/ai/repo-structure.md` — populated isolated e2e package and root-workspace relationship
- `docs/ai/tickets/GRF-232-e2e-suite.md`, `docs/ai/tickets/INDEX.md`, and this file — acceptance, status, and implementation record

**Files removed**

None.

**Contracts introduced or changed**

```ts
// e2e/tests/harness.ts
export const runtimeURL: string;
export const qdrantURL: string;
export async function buildImage(): Promise<void>;
export async function startFreshStack(collection: string): Promise<void>;
export async function stopStack(collection: string): Promise<void>;
export async function waitForHealth(timeoutMs?: number): Promise<void>;
export async function startRuntime(collection: string): Promise<void>;
export async function terminateRuntime(collection: string): Promise<{
	beforeWal: number;
	afterWal: number;
	exitCode: number;
}>;
export const test: TestType<PlaywrightTestArgs & { stack: { collection: string } }, PlaywrightWorkerArgs>;
```

```go
// runtime/internal/bootstrap/bootstrap.go
http.Server{
		BaseContext: func(net.Listener) context.Context { return ctx },
		// existing timeout fields unchanged
}
```

The isolated package commands are `node node_modules/@playwright/test/cli.js test`, `... test --repeat-each=3`, and `... install chromium`. It is not a member of the root pnpm workspace; install it with `pnpm install --ignore-workspace --frozen-lockfile`. Defaults are Runtime `127.0.0.1:18082`, Qdrant `127.0.0.1:16333`, image `gyrifi:e2e`, Qdrant `v1.13.4`, and deterministic evaluation.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep `e2e/` outside the root workspace with its own lockfile | Docker/Chromium qualification has a separate install and execution lifecycle and needs no Studio hoisting | Add it to `pnpm-workspace.yaml` | Every normal Studio install would carry browser-test package resolution without sharing useful dependencies |
| Build through Compose from the repository Dockerfile in global setup | Every suite must qualify the current embedded Studio and Runtime binary | Test `compose.yaml` with a previously published or manually tagged image | It can silently test stale code and miss packaging defects |
| Use one worker and recreate volumes per test | Fixed loopback ports remain simple while tests retain order independence and unique collections | Parallel workers against one Runtime/collection | Process-global collection configuration and shared ports would create cross-test races |
| Keep two high-value journeys | Browser/image tests are expensive and should prove boundaries, not duplicate component branches | Mirror the 144-test Studio suite in Chromium | It increases runtime and flakiness without stronger product evidence |
| Ingest Changes through HTTP but drive every supported governance action through Studio | This matches applications writing the durable inbox and operators governing in the browser | Create Proposals/evidence/approvals/releases directly by API | It would bypass the product controls the ticket exists to qualify |
| Assert Qdrant through raw REST | The suite proves the target mutation independently without broadening dependencies | Add a Qdrant client package or trust Studio state | A UI-only assertion cannot prove data landed, and a client adds unnecessary surface |
| Set `http.Server.BaseContext` to the signal context | Active SSE requests must observe SIGTERM before `Shutdown` waits | Increase the shutdown timeout or force-close all connections | A longer timeout still exits late; force close is not graceful and can interrupt ordinary requests |
| Make the collection initializer idempotent | Compose re-evaluates completed dependencies when only Gyrifi restarts | Recreate the collection during restart | It would destroy the target state that the persistence test must preserve |

**Deviations from the ticket**

GRF-233 has not created a CI workflow, so this ticket cannot yet make e2e failures block the pipeline or publish the retained artifacts there. The pinned package, stable `pnpm --ignore-workspace test` command, and `test-results/` artifact contract are ready for GRF-233 to invoke. The optional Gemma job was not created because there is neither a workflow nor an externally provisioned model; the required default suite runs with inference disabled exactly as specified. All other acceptance criteria were met.

**Traps for future work**

- `docker compose start gyrifi` re-evaluates its completed `qdrant-init` dependency. The initializer must remain idempotent or persistence restart fails on Qdrant's existing-collection response.
- `http.Server.Shutdown` does not cancel active request contexts by itself. Long-lived SSE handlers need the process cancellation propagated through `BaseContext`.
- Qdrant cosine collections normalize vectors. Browser qualification compares vector direction and exact payload, not raw vector bytes.
- Qdrant point IDs in this suite are valid numeric IDs; logical Change `unit` is the matching decimal string.
- Hash fragments never reach the server and cannot alone prove SPA fallback. The deep links intentionally request `/operator` before `#changes` or `#proposals/{id}` so the Go fallback serves `index.html` for an unmatched path.
- Playwright's default TypeScript transform needs this isolated package declared as ESM because the harness resolves Compose through `import.meta.url`.
- A `docker kill --signal TERM` while Studio is connected tests the real SSE shutdown path; a container with no browser connection will not reproduce the former exit-1 defect.

**Tests added**

- `e2e/tests/governance.spec.ts` empty-first-run journey — embedded Studio boot, empty call to action, direct empty Qdrant assertion, and UI Ledger creation on fresh volumes
- `e2e/tests/governance.spec.ts` full journey — rendered server gate reasons, two real target Releases, rollback to original point payload/vector direction, forward third HEAD, all durable records after restart, exit code 0, bounded/non-growing WAL, and fresh-page fallback routes
- `e2e/tests/harness.ts` fixture — every test receives a unique Qdrant collection and fresh named governance/target volumes with health-gated startup and cleanup

**Docs updated**

- `README.md` §Browser end-to-end qualification — installation, invocation, ports, artifacts, and repetition
- `docs/ai/tech-spec.md` §§1–2 and §§12–13 — Playwright stack, graceful request cancellation, e2e surface, and quality command
- `docs/ai/repo-structure.md` §§1 and 4 — isolated package structure and workspace decision
- `docs/ai/tickets/GRF-232-e2e-suite.md` — acceptance results and explicit GRF-233 CI deferral
- `docs/ai/tickets/INDEX.md` — GRF-232 marked Done
- `docs/ai/phases/phase-4.md` — ticket status and this completion record

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests (cached)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm coverage && pnpm build
Scope: all 2 workspace projects
Already up to date
Test Files  47 passed (47)
Tests       144 passed (144)
All files  | 86.20 % Stmts | 85.83 % Branch | 71.14 % Funcs | 86.20 % Lines
✓ 1867 modules transformed.
✓ built in 894ms

$ cd e2e && pnpm install --ignore-workspace --frozen-lockfile && pnpm --ignore-workspace test:repeat
Already up to date
Running 6 tests using 1 worker
✓ 1 a fresh shipped image shows the empty first-run path (6.0s)
✓ 2 the shipped image governs, rolls back, restarts, and deep-links (6.1s)
✓ 3 a fresh shipped image shows the empty first-run path (5.9s)
✓ 4 the shipped image governs, rolls back, restarts, and deep-links (6.2s)
✓ 5 a fresh shipped image shows the empty first-run path (5.8s)
✓ 6 the shipped image governs, rolls back, restarts, and deep-links (6.2s)
6 passed (45.9s)

$ docker build -t gyrifi:local .
[+] Building 3.4s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /out/gyrifi ./cmd/gyrifi
=> naming to docker.io/library/gyrifi:local

$ cd docs/ai/tickets && diff <ticket files> <INDEX status rows>
tickets consistent

$ git diff --check
(no output)
```

**Follow-ups discovered**

- GRF-233 must run `pnpm install --ignore-workspace --frozen-lockfile`, install Chromium with the direct entry point, invoke `pnpm --ignore-workspace test`, and upload `e2e/test-results/` on failure so the existing suite becomes a blocking check.
- An optional externally provisioned Gemma job can reuse this harness after CI has a model source; the required path must remain model-free.

### GRF-231 — Qdrant integration qualification

| | |
|---|---|
| Completed | 2026-09-01 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

The Qdrant adapter now has a build-tagged qualification suite against real, secured Qdrant 1.13.4. It verifies metric-specific vector behavior, complete JSON payload round trips, idempotency, deletes, classified failures, drift, visible partial application, and a full Engine release/before-image/forward-rollback flow. Integration testing exposed and fixed numeric point deletion: Qdrant rejects a logical numeric unit serialized as a JSON string, so the adapter now preserves or reconstructs the point ID's JSON type.

CI now starts an immutable Qdrant 1.13.4 service and runs the integration suite twice under the race detector as a normal failure-enforcing job. Browser E2E CI remains disabled exactly as required by the GRF-233 extension-point decision.

**Files added**

- `runtime/internal/targets/qdrant/integration_test.go` — real-service harness, isolated collection lifecycle, behavior qualification, and Engine release/rollback coverage

**Files changed**

- `runtime/internal/targets/target.go` — stable not-found, semantic, unavailable, and authentication classifications
- `runtime/internal/targets/qdrant/qdrant.go` — classified REST failures and numeric/UUID-correct delete IDs
- `runtime/internal/targets/qdrant/qdrant_test.go` — fast classification and credential non-disclosure coverage
- `.github/workflows/ci.yml` — required secured-Qdrant integration job pinned by digest; E2E remains disabled
- `AGENTS.md` — current workflow enforcement statement
- `docs/ai/tech-spec.md` — verified version, metric/payload findings, partial-Plan semantics, classifications, invocation, CI, tests, and gap closure
- `docs/ai/repo-structure.md` — integration test and workflow responsibilities
- `docs/ai/tickets/GRF-231-qdrant-qualification.md` and `INDEX.md` — acceptance and completion bookkeeping
- `docs/ai/phases/phase-4.md` — phase status, exit criteria, and this completion record

**Files removed**

None.

**Contracts introduced or changed**

```go
var (
	ErrNotFound       = errors.New("target resource not found")
	ErrSemantic       = errors.New("target rejected the operation")
	ErrUnavailable    = errors.New("target is unavailable")
	ErrAuthentication = errors.New("target authentication failed")
)
```

Qdrant REST classification is now observable through `errors.Is`: 404 collection failures map to `targets.ErrNotFound`; 400/409/422 responses map to `targets.ErrSemantic`; 401/403 map to `targets.ErrAuthentication`; transport and other non-success responses map to `targets.ErrUnavailable`. API keys and response bodies never enter these error messages.

```text
integration build tag: integration
GYRIFI_TEST_QDRANT_URL=http://127.0.0.1:6333
GYRIFI_TEST_QDRANT_API_KEY=gyrifi-integration-key
CI job name: Qdrant integration
CI image: qdrant/qdrant@sha256:318c11b72aaab96b36e9662ad244de3cabd0653a1b942d4e8191f18296c81af0 (v1.13.4)
```

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Use the existing REST adapter plus standard-library harness helpers | Qualification must exercise the production wire path without introducing a second client implementation | Add a Qdrant Go client dependency for setup/assertions | It would test a different stack and violate the ticket's no-client-library instruction |
| Secure the CI service and use a fixed non-production test key | Authentication and non-disclosure are continuously exercised, not conditionally skipped in CI | Run an unsecured service and test auth only locally | The required job would leave a stated failure mode unqualified |
| Keep cosine direction tolerance at `1e-6` | Real Qdrant normalisation of `[3,4,0]` passed while payload drift remained detectable | Loosen tolerance when adding integration tests | No empirical failure justified weakening verification |
| Classify statuses with exported target sentinels | GRF-221 can distinguish invalid Change content from retryable infrastructure without importing Qdrant | Parse adapter error strings | Text parsing is unstable and risks turning infrastructure errors into governance decisions |
| Document sequential partial application rather than claim batch atomicity | A valid first point remains when a later wrong-dimension point is rejected; the returned semantic error makes the partial outcome visible and Engine enters recovery | Attempt compensating writes inside `Apply` | Compensation can fail and would forge a stronger atomicity guarantee than Qdrant provides |

**Deviations from the ticket**

None. Every acceptance criterion was exercised against Qdrant 1.13.4, including two consecutive clean race-enabled runs. `docs/ai/product.md` already had no Qdrant-fake gap row, so no product text required removal; the stale technical gap was removed from `tech-spec.md` §14.

**Traps for future work**

- Qdrant numeric point IDs must be JSON numbers in delete selectors. The string `"1"` is treated as an invalid UUID and returns HTTP 400.
- The adapter's Plan is not one Qdrant batch: each operation is a separate synchronous request. Earlier operations can land before a later semantic rejection; Release recovery is the owning safety mechanism.
- The integration file defines `TestMain`; when the URL is configured, it waits for Qdrant, logs the live version, and sweeps stale `gyrifi_it_*` collections before tests. Keep every active test collection unique and cleanup-owned.
- Cosine read fingerprints differ bytewise from pre-write desired values because Qdrant normalises vectors. Release verification must continue comparing vector direction, not desired/read hash equality.

**Tests added**

- `runtime/internal/targets/qdrant/integration_test.go` — Cosine/Dot/Euclid and all JSON payload types; identical PUT; existing/absent DELETE; missing collection, wrong dimension, unreachable service, and auth classifications; drift mismatch; partial Plan outcome; complete release, retained before-image, rollback, and final target value
- `runtime/internal/targets/qdrant/qdrant_test.go` — fake-server status classification, transport classification, and API-key exclusion from errors

**Docs updated**

- `docs/ai/tech-spec.md` §§9, 12, 13, and 14 — real Qdrant evidence, invocation, CI, and gap closure
- `docs/ai/repo-structure.md` §§1 and 4 — package and workflow structure
- `AGENTS.md` §5 — enabled integration gate and disabled E2E extension
- `docs/ai/tickets/GRF-231-qdrant-qualification.md`, `INDEX.md`, and this file — completion records

**Verification**

```text
$ test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests (cached)

$ GYRIFI_TEST_QDRANT_URL=http://127.0.0.1:16333 GYRIFI_TEST_QDRANT_API_KEY=gyrifi-integration-key go test -tags=integration ./internal/targets/qdrant -race -count=1 -v
2026/09/01 04:05:39 Qdrant integration version: 1.13.4
--- PASS: TestIntegrationRoundTripMetricsAndPayloadTypes (Cosine, Dot, Euclid)
--- PASS: TestIntegrationDeleteExistingAndAbsent
--- PASS: TestIntegrationFailureClassifications
--- PASS: TestIntegrationDriftAndPartialPlanFailure
--- PASS: TestIntegrationEngineReleaseAndRollback
PASS
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 1.962s

$ <same integration command, second consecutive run>
2026/09/01 04:05:41 Qdrant integration version: 1.13.4
PASS
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 1.970s

$ pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Already up to date
Test Files  48 passed (48)
Tests  153 passed (153)
✓ 1868 modules transformed.
✓ built in 796ms

$ docker build -t gyrifi:local .
[+] Building 30.7s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build ... 27.0s
=> => naming to docker.io/library/gyrifi:local

$ ticket index consistency check
tickets consistent
```

**Follow-ups discovered**

Named vectors, sparse vectors, and Qdrant multitenancy remain intentionally unsupported. Integration tests did not reveal a defect requiring a new ticket. Browser E2E CI remains disabled per the owner's explicit GRF-233 direction.
