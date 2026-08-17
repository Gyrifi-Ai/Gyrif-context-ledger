# Repository structure

Status: reflects the tree as it exists. Update this file whenever you add a directory or change a layering rule.

---

## 1. Top level

```text
gyrif-context-ledger/
├── AGENTS.md                  # working agreement for AI agents — read first
├── README.md                  # product README for humans
├── Dockerfile                 # 4-stage build, produces the only shipping artifact
├── LICENSE
├── package.json               # pnpm workspace root, thin script proxies
├── pnpm-workspace.yaml        # packages: [studio]
├── pnpm-lock.yaml
├── .dockerignore / .gitignore
├── docs/
│   ├── adr/                   # architecture decision records
│   ├── ai/                    # the documentation set in this folder
│   └── archive/               # SUPERSEDED docs — historical only, never authoritative
├── e2e/                       # browser end-to-end suite (currently empty — GRF-232)
├── runtime/                   # the single Go module
└── studio/                    # the single frontend package
```

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
│   ├── bootstrap/bootstrap.go       # composition root + lifecycle; const Version
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
│   │   ├── changes.go               #   CreateChange + idempotency
│   │   ├── proposals.go             #   CreateProposal, ApproveProposal
│   │   ├── evaluation.go            #   EvaluateProposal
│   │   ├── releases.go              #   ReleaseProposal, RecoverReleases
│   │   └── rollback.go              #   CreateRollbackProposal
│   ├── repository/                  # Gyrifi-owned persistence only
│   │   ├── repository.go            #   Repository interface + sentinel errors
│   │   ├── sqlite.go                #   OpenSQLite, pragmas, all SQL, transactions
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
│           └── static/              #   embedded Studio assets (index.html fallback in git)
├── migrations/
│   ├── 001_initial.sql              # full schema
│   └── migrations.go                # //go:embed + ordered application
└── tests/
    └── change_flow_test.go          # end-to-end governance flow against fakes
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
├── package.json
├── tsconfig.json               # strict: true, target ES2022, moduleResolution Bundler, noEmit, paths @/* → src/*
├── vite.config.ts              # port 5173; proxies /api and /events → 127.0.0.1:18080; @tailwindcss/vite plugin
├── index.html                  # <div id="root">, theme-color #06080c
└── src/
    ├── main.tsx                # imports styles.css, calls bootstrap(root)
    ├── styles.css              # Tailwind v4 entry: @theme tokens (design-system palette) + base layer
    ├── lib/utils.ts            # cn() — clsx + tailwind-merge
    ├── app/
    │   ├── bootstrap.tsx       # createRoot + StrictMode + Providers + Shell
    │   ├── providers.tsx       # AppState context: { ledgerId, setLedgerId } persisted to localStorage
    │   ├── router.tsx          # hash routing: Route union + useRoute()
    │   └── shell.tsx           # maps Route → page component
    ├── api/
    │   ├── types.ts            # the API contract types (mirror of Go JSON tags)
    │   ├── client.ts           # request<T>() + the `api` object
    │   ├── client.test.ts
    │   └── events.ts           # subscribeToEvents() over EventSource
    ├── components/ui/          # shadcn/ui components (button, card, badge, input, textarea, label,
    │                           #  table, dialog, checkbox, separator, skeleton, tooltip)
    ├── ui/                     # VISUAL ONLY. must not know about ledgers/changes/etc.
    │   ├── patterns/status-badge.tsx
    │   ├── layout/application-shell.tsx
    │   └── feedback/empty-state.tsx
    ├── features/               # ALL domain-aware UI
    │   ├── shared/status.ts    # domain status → badge tone mapping (design-system §2.2)
    │   ├── ledgers/ledgers-page.tsx
    │   ├── changes/changes-page.tsx
    │   ├── proposals/proposals-page.tsx
    │   └── releases/releases-page.tsx
    └── test/                   # currently empty — GRF-230
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

---

## 4. Build and packaging

`Dockerfile` has four stages:

| Stage | Base | Does |
|---|---|---|
| `studio-build` | `node:24-bookworm-slim` | installs pnpm 11.15.1, `pnpm install --frozen-lockfile`, `pnpm --filter @gyrifi/studio build` → `studio/dist` |
| `llama-runtime` | `ghcr.io/ggml-org/llama.cpp:server` | source of `llama-server` and its native libs |
| `runtime-build` | `golang:1.24-bookworm` | copies `studio/dist` into `internal/interfaces/http/static/`, runs `go test ./...`, builds `CGO_ENABLED=0` binary with `-trimpath -ldflags="-s -w"` |
| `runtime` | `ubuntu:24.04` | minimal libs, non-root user `gyrifi` (uid/gid 10001), `VOLUME /data`, `EXPOSE 8080`, entrypoint `/usr/local/bin/gyrifi` |

Node and Go are **build-stage only**. The final image contains one Go binary plus llama-server artifacts.

`pnpm` scripts in `studio/package.json` invoke tool entry points directly (`node node_modules/vite/bin/vite.js`) because the local workspace path contains a `:`, which makes prepending `node_modules/.bin` to `PATH` unsafe. **Keep this pattern** when adding scripts.

---

## 5. Files you must not casually touch

| Path | Why |
|---|---|
| `runtime/migrations/001_initial.sql` | already applied in existing installs; add a new numbered migration instead |
| `runtime/internal/interfaces/http/static/index.html` | the committed fallback page; the rest of that directory is gitignored build output |
| `pnpm-lock.yaml` | only change via `pnpm install`; the Docker build uses `--frozen-lockfile` |
| `docs/archive/**` | historical record; do not "fix" it |
