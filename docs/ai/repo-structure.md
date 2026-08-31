# Repository structure

Status: reflects the tree as it exists. Update this file whenever you add a directory or change a layering rule.

---

## 1. Top level

```text
gyrif-context-ledger/
├── AGENTS.md                  # working agreement for AI agents — read first
├── README.md                  # product README for humans
├── Dockerfile                 # 4-stage build, produces the only shipping artifact
├── compose.yaml               # local Gyrifi + Qdrant launch, loopback-only Studio/API port
├── LICENSE
├── package.json               # pnpm workspace root, thin script proxies
├── pnpm-workspace.yaml        # packages: [studio]
├── pnpm-lock.yaml
├── .dockerignore / .gitignore
├── .github/
│   └── workflows/ci.yml       # push/PR Runtime, Studio, coverage, and image quality gate
├── .vscode/
│   └── launch.json            # Run and Debug entry point for the local Compose stack
├── docs/
│   ├── adr/                   # architecture decision records
│   ├── ai/                    # the documentation set in this folder
│   └── archive/               # SUPERSEDED docs — historical only, never authoritative
├── e2e/                       # isolated Playwright package; built-image browser qualification
├── runtime/                   # the single Go module
└── studio/                    # the single frontend package
```

`e2e/` is deliberately not listed in the root `pnpm-workspace.yaml`: it is an expensive Docker/Chromium qualification package with its own lockfile and install lifecycle, and it needs no dependency hoisting from Studio.

---

## 2. `runtime/` — Go modular monolith

Module: `github.com/gyrifi/gyrif-context-ledger/runtime`, Go 1.24.
Only direct dependency: `modernc.org/sqlite` (pure Go, no cgo).

```text
runtime/
├── go.mod / go.sum
├── cmd/
│   └── gyrifi/
│       └── main.go                  # signal context, calls bootstrap.Run(ctx, os.Args[1:])
├── internal/
│   ├── buildinfo/                   # linker-injected Version, Commit, Date + stable String
│   ├── bootstrap/bootstrap.go       # composition root + lifecycle
│   ├── config/config.go             # env → Config, loaded exactly once
│   ├── ledger/                      # PURE domain. no I/O, ever.
│   │   ├── ledger.go                #   Ledger, Head
│   │   ├── change.go                #   Change, ChangeAction, ChangeStatus
│   │   ├── proposal.go              #   Proposal, ProposalStatus, CheckResult, Approval
│   │   ├── release.go               #   Release, ReleaseIntent, ReleaseIntentStatus
│   │   ├── identity.go              #   NewID, Hash, Fingerprint
│   │   ├── invariants.go            #   ValidateChange, ProposalHash, ReleaseHash, sentinel errors
│   │   └── invariants_test.go
│   ├── engine/                      # application facade. one Engine, shared by HTTP and CLI.
│   │   ├── engine.go                #   Engine struct, ErrorCode, Error, wrap, PublicError, list methods
│   │   ├── events.go                #   typed advisory events + non-blocking in-process Broker
│   │   ├── events_test.go           #   delivery, slow-subscriber, unsubscribe concurrency
│   │   ├── changes.go               #   CreateChange + idempotency
│   │   ├── proposals.go             #   CreateProposal, ApproveProposal
│   │   ├── proposal_detail.go        #   detail/evidence reads + server-authoritative gates
│   │   ├── evaluation.go            #   EvaluateProposal
│   │   ├── releases.go              #   ReleaseProposal, RecoverReleases
│   │   ├── release_intents.go        #   Intent inspection, verification-only retry, explicit abandonment
│   │   └── rollback.go              #   CreateRollbackProposal
│   ├── repository/                  # Gyrifi-owned persistence only
│   │   ├── repository.go            #   Repository interface + sentinel errors
│   │   ├── sqlite.go                #   OpenSQLite, pragmas, all SQL, transactions
│   │   ├── sqlite_test.go            #   check/approval ordering and empty-list contracts
│   │   └── objects.go               #   ObjectStore: sha256 CAS, 2-char shard, temp+fsync+rename
│   ├── targets/
│   │   ├── target.go                #   TargetAdapter interface, Plan, Operation, Value, Preview, Capabilities
│   │   └── qdrant/
│   │       ├── qdrant.go            #   the only adapter
│   │       └── qdrant_test.go
│   ├── inference/
│   │   ├── provider.go              #   Provider interface, EvaluationRequest/Result, Finding
│   │   ├── llamacpp.go              #   StartLlamaServer, child process supervision, HTTP provider
│   │   └── llamacpp_test.go
│   └── interfaces/                  # protocol translation only. no business rules.
│       ├── cli/cli.go               #   doctor, version
│       └── http/
│           ├── server.go            #   routes, middleware, encoding, SSE, SPA fallback
│           ├── events_test.go       #   SSE forwarding, Ledger filtering, cancellation
│           └── static/              #   embedded Studio assets (index.html fallback in git)
├── migrations/
│   ├── 001_initial.sql              # full schema
│   ├── 002_release_intent_resolution.sql # additive operator-resolution fields
│   └── migrations.go                # //go:embed + ordered application
└── tests/
    ├── change_flow_test.go          # end-to-end governance flow against fakes
    ├── proposal_detail_test.go      # detail gates, evidence, scoping, and HTTP contracts
    └── release_recovery_test.go     # Intent reads, retry/resolve, release guard, before-images
```

### Layering rules — hard

```text
interfaces ──► engine ──► ledger
                 ├──────► repository (interface)
                 ├──────► targets    (interface)
                 └──────► inference  (interface)
bootstrap ──► everything (composition only)
```

| Rule | |
|---|---|
| `ledger` MUST NOT import anything but the standard library | it is the pure model |
| `ledger` MUST NOT reference HTTP, SQL, files, env vars, Qdrant, llama.cpp, or React | |
| `engine` MUST NOT contain SQL strings, file paths, or HTTP status codes | |
| `engine` MUST NOT import `targets/qdrant` or `inference/llamacpp` concretely — only the interfaces | |
| `repository` MUST own all SQL and all filesystem access for Gyrifi state | |
| `targets/qdrant` MUST own every Qdrant request/response type | |
| `inference` MUST return typed evidence and MUST NOT touch `Repository` or `TargetAdapter` | |
| `interfaces/*` MUST only decode requests, call one Engine method, and encode the result | |
| Only `engine/releases.go` may call `TargetAdapter.Apply` / `Restore` | |
| `bootstrap` is the only place that constructs concrete implementations | no global service locator |

### Where does new backend code go?

| You are adding… | Put it in |
|---|---|
| A new domain concept, state, or hash | `internal/ledger/` |
| A new business rule or workflow step | `internal/engine/<area>.go` |
| A new query or table access | `internal/repository/sqlite.go` + a method on `Repository` |
| A new column or table | a **new** `migrations/00N_*.sql`, never edit `001_initial.sql` |
| A new HTTP endpoint | `internal/interfaces/http/server.go` `routes()` + a handler + an Engine method |
| Anything Qdrant-specific | `internal/targets/qdrant/` |
| Anything model/prompt-specific | `internal/inference/` |
| Wiring, lifecycle, env | `internal/bootstrap/` or `internal/config/` |

---

## 3. `studio/` — React frontend

Package: `@gyrifi/studio`. React 19.1.1, TypeScript 5.9.2 (strict), Vite 7.1.4, Vitest 3.2.4. UI is built with **shadcn/ui conventions on Tailwind CSS v4** (`tailwindcss` + `@tailwindcss/vite`), Radix UI primitives, `lucide-react` icons, and `class-variance-authority`/`clsx`/`tailwind-merge` (stack authorised 2026-08-17, see `design-system.md` §8). No router library, no data-fetching library.

```text
studio/
├── package.json                 # direct-entry dev/build/typecheck/test/coverage scripts
├── tsconfig.json               # strict: true, target ES2022, moduleResolution Bundler, noEmit, paths @/* → src/*
├── vite.config.ts              # Vite/Tailwind, dev proxy, jsdom test setup, V8 coverage thresholds
├── index.html                  # <div id="root">, theme-color #06080c
└── src/
    ├── main.tsx                # imports styles.css, calls bootstrap(root)
    ├── vite-env.d.ts           # Vite client environment declarations
    ├── styles.css              # Tailwind v4 entry: @theme tokens (design-system palette) + base layer
    ├── lib/utils.ts            # cn() — clsx + tailwind-merge
    ├── app/
    │   ├── bootstrap.tsx       # createRoot + StrictMode + Providers + root ErrorBoundary + Shell
    │   ├── error-boundary.tsx  # resettable root/section React render boundary
    │   ├── providers.tsx       # Reachability + AppState providers; selected Ledger id and switcher-open requests
    │   ├── reachability.tsx    # Runtime polling, request health, stream state, reconnect invalidation
    │   ├── reachability-banner.tsx # persistent application-level transport failure surface
    │   ├── router.tsx          # hash routing: Route union + useRoute()
    │   ├── use-async.ts        # dependency-free useQuery/useMutation lifecycle primitives
    │   ├── use-ledger-events.ts # registers active-query invalidation for stream reconnect/domain events
    │   └── shell.tsx           # maps Route → section-boundary-wrapped page component
    ├── api/
    │   ├── types.ts            # the API contract types (mirror of Go JSON tags)
    │   ├── client.ts           # request<T>() + the `api` object
    │   ├── client.test.ts
    │   ├── events.ts           # stateful EventSource subscription with bounded CLOSED retries
    │   └── events.test.ts      # stream state, retry ceiling, manual reconnect, teardown
    ├── components/ui/          # shadcn/ui components plus focused primitive composition tests
    ├── ui/                     # VISUAL ONLY. must not know about ledgers/changes/etc.
    │   ├── primitives/         # Button, field controls, segmented control, domain-free SVG icons
    │   ├── patterns/           # DataTable, badges, hashes, code, timeline, stats, confirmation dialog
    │   ├── layout/             # slot-based application shell, page header, panel, drawer
    │   └── feedback/           # loading skeleton, empty/error states, AsyncBoundary query state renderer
    ├── features/               # ALL domain-aware UI
    │   ├── shared/             # domain status mapping and relative-time formatting
    │   ├── shell/              # Ledger switcher, HEAD chip, navigation, and shared connection status
    │   ├── ledgers/ledgers-page.tsx
    │   ├── changes/            # inbox page, pure submission/filter/order helpers, selection action bar, tests
    │   ├── proposals/         # review queue/detail, progress, evidence, approval, release, ordered creation,
    │   │                      # server-gate projections, presets, and focused tests
    │   └── releases/          # timeline page, plan drawer, recovery banner/actions, rollback dialog, focused tests
    └── test/
        ├── setup.ts            # jest-dom, browser shims, cleanup, console-error guard, API reset
        ├── render.tsx          # renderWithProviders() + userEvent
        └── api-mock.ts         # typed fetch router derived from the real Api surface
```

### Frontend rules — hard

| Rule | |
|---|---|
| `ui/` MUST be domain-agnostic | a component in `ui/` may not import from `features/` or `api/types` domain types |
| `features/` owns all domain-aware UI and all API calls | |
| `api/` is the only place that calls `fetch` | |
| `app/` owns routing, providers, and mounting — nothing else | |
| Governance rules MUST NOT be reimplemented in the frontend | e.g. never decide client-side whether a Proposal is releasable; ask the server |
| No new runtime dependency without an ADR | the current runtime dependency list is exactly `react`, `react-dom` |

### Where does new frontend code go?

| You are adding… | Put it in |
|---|---|
| A reusable visual element with no domain meaning | `ui/primitives/` |
| A reusable composition of primitives (badge, stat, key-value) | `ui/patterns/` |
| Page chrome, grids, panels | `ui/layout/` |
| Loading, empty, error surfaces | `ui/feedback/` |
| A screen or any domain widget | `features/<area>/` |
| A new endpoint call or contract type | `api/client.ts` and `api/types.ts` |
| A design token, colour, or spacing value | `styles.css` `:root` — never inline hex in a component |

Test files are co-located as `*.test.ts` or `*.test.tsx`; only shared test infrastructure belongs in `src/test/`. Tests query roles, labels, and visible text rather than treating utility classes as behavior. `pnpm coverage` enforces the repository's 80% statement and 75% branch thresholds.

---

## 4. Build and packaging

### Continuous integration

`.github/workflows/ci.yml` runs on pushes and pull requests with workflow-level `contents: read` permission and ref-scoped cancellation. Runtime and Studio execute independently; the image job starts only after both pass. Every external action is pinned to a full commit SHA.

The Runtime job uses Go 1.24 from `runtime/go.mod` and enforces gofmt without rewriting files, a clean `go mod tidy`, `go vet`, race-enabled tests, and `go build`. The Studio job pins Node 24 and pnpm 11.15.1, installs the root workspace with `--frozen-lockfile`, and invokes the direct-entry `typecheck`, `test`, `coverage`, and `build` scripts. The image job uses Buildx cache, accepts `VERSION`, `COMMIT`, and `BUILD_DATE` build arguments, smoke-tests the loaded image through the system status endpoint, and uploads a one-day Docker image artifact.

Qdrant integration and browser e2e jobs are retained as disabled extension points. Their owning qualification work must enable them and make them required checks; ordinary CI does not silently run or waive either surface.

### Browser qualification package

```text
e2e/
├── package.json               # direct-entry Playwright test, repeat, and Chromium-install scripts
├── pnpm-lock.yaml             # isolated, pinned test dependency graph
├── playwright.config.ts       # Chromium-only, one worker, bounded timeout, failure traces/screenshots
├── compose.yaml               # current image build + pinned Qdrant + per-run named volumes/collection
└── tests/
    ├── global-setup.ts        # builds the repository Dockerfile and waits for healthy Runtime startup
    ├── global-teardown.ts     # removes containers, network, and qualification volumes
    ├── harness.ts             # Compose, health, SIGTERM/WAL, ingestion, and Qdrant helpers
    └── governance.spec.ts     # empty first run and full release/rollback/restart/deep-link journey
```

Run installs inside this package with `pnpm --ignore-workspace`; this prevents pnpm from walking up to the root workspace. Tests use only roles and accessible names, never selectors coupled to styling or DOM position. The harness calls Qdrant REST directly and does not add a target client library.

`Dockerfile` has four stages:

| Stage | Base | Does |
|---|---|---|
| `studio-build` | `node:24-bookworm-slim` | installs pnpm 11.15.1, `pnpm install --frozen-lockfile`, `pnpm --filter @gyrifi/studio build` → `studio/dist` |
| `llama-runtime` | `ghcr.io/ggml-org/llama.cpp:server` | source of `llama-server` and its native libs |
| `runtime-build` | `golang:1.24-bookworm` | copies `studio/dist` into `internal/interfaces/http/static/`, runs `go test ./...`, builds `CGO_ENABLED=0` binary with `-trimpath -ldflags="-s -w"` |
| `runtime` | `ubuntu:24.04` | minimal libs, non-root user `gyrifi` (uid/gid 10001), `VOLUME /data`, `EXPOSE 8080`, entrypoint `/usr/local/bin/gyrifi` |

Image builds accept `VERSION`, `COMMIT`, and `BUILD_DATE`, each with development-safe defaults. The Runtime build preserves `CGO_ENABLED=0`, `-trimpath`, and `-s -w` while injecting those values into `internal/buildinfo`. The final image repeats them as OCI version, revision, and creation labels alongside the source URL and `AGPL-3.0-only` license.

Node and Go are **build-stage only**. The final image contains one Go binary plus llama-server artifacts.

`pnpm` scripts in `studio/package.json` invoke tool entry points directly (`node node_modules/vite/bin/vite.js`) because the local workspace path contains a `:`, which makes prepending `node_modules/.bin` to `PATH` unsafe. **Keep this pattern** when adding scripts.

### Local Compose launch

`compose.yaml` is development-only composition, not a second shipping artifact. It starts the built Gyrifi image, a pinned Qdrant target, and an idempotent collection initializer. The only published port is `127.0.0.1:8080`; both Gyrifi and Qdrant use named Docker volumes. `.vscode/launch.json` runs `docker compose up --build --remove-orphans` from the workspace root through the Run and Debug control. Do not change this binding to all interfaces before GRF-220 adds authentication.

---

## 5. Files you must not casually touch

| Path | Why |
|---|---|
| `runtime/migrations/001_initial.sql` | already applied in existing installs; add a new numbered migration instead |
| `runtime/internal/interfaces/http/static/index.html` | the committed fallback page; the rest of that directory is gitignored build output |
| `pnpm-lock.yaml` | only change via `pnpm install`; the Docker build uses `--frozen-lockfile` |
| `docs/archive/**` | historical record; do not "fix" it |
