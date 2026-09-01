# Phase 3 — Production hardening

**Goal:** make Gyrifi safe to run somewhere other than a developer's laptop. Everything here is a property the product currently lacks that would make a real deployment irresponsible.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-223](../tickets/GRF-223-build-metadata.md) | Build metadata and version consistency | S | — | Done |
| [GRF-221](../tickets/GRF-221-change-preparation.md) | Asynchronous Change preparation and base fingerprint | L | — | Done |
| [GRF-222](../tickets/GRF-222-retention-backup.md) | Retention budgets, quotas, and backup command | L | — | Not started |
| [GRF-224](../tickets/GRF-224-health-and-metrics.md) | Health, readiness, and operational metrics | M | — | Done |
| [GRF-225](../tickets/GRF-225-inference-supervision.md) | Inference process supervision | M | — | Done |
| [GRF-220](../tickets/GRF-220-authentication.md) | Trusted deployment boundary decision | XL | — | Done |
| [GRF-226](../tickets/GRF-226-rate-limiting.md) | Request rate limiting and abuse controls | M | — | Not started |
| [GRF-227](../tickets/GRF-227-local-docker-launch.md) | Local Docker launch | M | — | Done |

Listed in recommended execution order, smallest and least entangled first.

## Phase-level notes

- **ADR 0002 accepts a trusted deployment boundary instead of application authentication.** Gyrifi may bind to a company-controlled private interface only behind VM/VPC/VPN/service-mesh or reverse-proxy admission controls; public exposure is unsupported.
- GRF-221 makes the `ACCEPTED` and `INVALID` change statuses real for the first time. Phase 1's Changes inbox will need updating to render them — that update is inside GRF-221's scope, not a Phase 1 regression.
- Migration numbers continue from Phase 2: 005 is available because GRF-220 creates no schema; the next schema ticket takes it. Adjust and record here if the order changes.
- No authentication dependency is added. No remaining Phase 3 ticket adds a dependency.

## The theme

Each ticket in this phase closes a way the product can fail silently:

| Ticket | Silent failure it prevents |
|---|---|
| GRF-223 | A bug report that cannot be tied to a build |
| GRF-221 | A Change whose relationship to the target's actual state is unknown |
| GRF-222 | A volume filling up mid-release, taking rollback material with it |
| GRF-224 | An orchestrator that cannot tell a healthy runtime from a broken one |
| GRF-225 | An inference process that died hours ago and took its logs with it |
| GRF-220 | An undocumented trust boundary that invites unsafe public exposure |
| GRF-226 | One runaway client starving every operator out of the system |

## Exit criteria

- [ ] All seven tickets complete.
- [ ] The runtime reports one accurate, build-injected version everywhere.
- [ ] Every Change has a known relationship to the target's observed state before it can be proposed.
- [ ] Disk growth is bounded and a supported backup/restore procedure exists and is tested.
- [x] The trusted VM/VPC deployment boundary and its lack of application-authenticated identity are documented without ambiguity.
- [ ] Liveness, readiness, and metrics are exposed, and a `RECOVERY_REQUIRED` intent is visible without making the runtime unready.
- [x] The inference child process is supervised, its output is captured, and its failures are legible.
- [ ] No single client can starve the runtime.
- [ ] `go test ./...` green with `-race`; `docker build` green.

## Completed entries

### GRF-227 — Local Docker launch

| | |
|---|---|
| Completed | 2026-08-17 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

The repository now has a single local Compose launch that builds the shipping Gyrifi image, starts a local Qdrant target, and provisions the required development collection. A Run and Debug entry launches that same Compose command, so no separate Go, Studio, Node, or Qdrant process is required. The application port remains bound exclusively to loopback because runtime authentication has not landed.

**Files added**

- `compose.yaml` — local Gyrifi, Qdrant, and idempotent collection-initializer topology
- `.vscode/launch.json` — Run and Debug entry for the local Compose stack
- `docs/ai/tickets/GRF-227-local-docker-launch.md` — scoped work order and acceptance record

**Files changed**

- `README.md` — primary one-action launch, shutdown, reset, and local-only guidance
- `docs/ai/tech-spec.md` — local Compose contract and data/lifecycle semantics
- `docs/ai/repo-structure.md` — Compose and VS Code launch files in the current tree
- `docs/ai/tickets/INDEX.md` — GRF-227 registration and completion status
- `docs/ai/phases/phase-3.md` — Phase 3 ticket registration and this completion record

**Files removed**

None.

**Contracts introduced or changed**

```yaml
# compose.yaml
gyrifi:
	environment:
		GYRIFI_QDRANT_URL: http://qdrant:6333
		GYRIFI_QDRANT_COLLECTION: gyrifi
	ports:
		- "127.0.0.1:8080:8080"
```

`qdrant-init` creates `{ "vectors": { "size": 3, "distance": "Cosine" } }` at `/collections/gyrifi` only after a missing-collection response. `.vscode/launch.json` exposes `Gyrifi: Start local stack`, executing `docker compose up --build --remove-orphans` with `${workspaceFolder}` as its working directory.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Compose starts Qdrant alongside Gyrifi | The adapter requires a Qdrant endpoint, so a usable local release workflow needs both services. | Bundle Qdrant into the Gyrifi image | Violates the one-process image design and makes state/lifecycle opaque. |
| Create the collection in a short-lived initializer | The adapter deliberately does not own target provisioning; first run must still be usable. | Unconditional collection `PUT` | It may replace a developer's existing local collection. |
| Bind Studio/API to `127.0.0.1` | GRF-220 has not added authentication. | Publish `8080` on all interfaces | An unauthenticated audit system must not be reachable from the network. |

**Deviations from the ticket**

None.

**Traps for future work**

The default Compose collection has vector size 3 solely to match the README's local example. It is a development convenience, not a configurable per-Ledger target contract. Keep the initializer non-destructive: an existing collection must never be replaced at startup.

**Tests added**

- None — no application behaviour changed. Compose model, first-start status, collection configuration, and restart persistence were validated against real containers.

**Docs updated**

- `README.md` §Quick start — one-action local startup and lifecycle
- `docs/ai/tech-spec.md` §1–§2 — local Docker Compose contract
- `docs/ai/repo-structure.md` §1 and §4 — new root and VS Code files
- `docs/ai/tickets/INDEX.md` — GRF-227 status

**Verification**

```
$ docker compose config
services.gyrifi.ports: 127.0.0.1:8080:8080
services.gyrifi.depends_on.qdrant-init.condition: service_completed_successfully

$ docker compose up --build -d
Container gyrif-context-ledger-qdrant-init-1 Exited
Container gyrif-context-ledger-gyrifi-1 Started

$ curl --fail --silent http://127.0.0.1:8080/api/v1/system/status
{"inference":"disabled","status":"ok","version":"0.1.0"}

$ curl http://qdrant:6333/collections/gyrifi
{"result":{"status":"green","config":{"params":{"vectors":{"size":3,"distance":"Cosine"}}}},"status":"ok",...}

$ docker compose down && docker compose up -d
$ docker compose logs qdrant-init
qdrant-init-1 | Qdrant collection gyrifi already exists.

$ go vet ./...
$ go test ./... -race
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests (cached)
$ go build ./...

$ pnpm typecheck
$ pnpm test
Test Files  1 passed (1)
Tests  2 passed (2)
$ pnpm build
✓ built in 1.14s

$ docker build -t gyrifi:local .
[+] Building 1.8s (31/31) FINISHED
=> => naming to docker.io/library/gyrifi:local

$ ticket index consistency check
tickets consistent
```

**Follow-ups discovered**

GRF-231 remains responsible for qualifying the Qdrant adapter and pinning its integration-test service version in CI; this local convenience stack does not replace that coverage.

### GRF-223 — Build metadata and version consistency

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | `b09a8d0`; branch push [run 33431187575](https://github.com/Gyrifi-Ai/Gyrif-context-ledger/actions/runs/33431187575) succeeded |
| Deviated from ticket | No |

**What was built**

The Runtime now has one build metadata source with development-safe defaults and deterministic linker injection. The CLI, system-status response, startup log, Studio shell footer, and OCI image labels all report values derived from that source. CI passes the same version, commit, and build date into the image and requires the live status response to report the injected version exactly.

**Files added**

- `runtime/internal/buildinfo/buildinfo.go` — process-wide `Version`, `Commit`, and `Date` variables plus the stable formatter
- `runtime/internal/buildinfo/buildinfo_test.go` — default and injected-value formatter coverage
- `runtime/internal/interfaces/cli/cli_test.go` — exact CLI version-output coverage
- `runtime/tests/buildinfo_test.go` — system-status build metadata integration coverage

**Files changed**

- `Dockerfile` — build arguments, linker injection, and OCI image labels while preserving static compilation and stripping
- `.github/workflows/ci.yml` — exact linker-injected image version assertion
- `runtime/internal/bootstrap/bootstrap.go` — removed the independent version constant, dispatches version before storage initialisation, and logs `buildinfo.String()`
- `runtime/internal/interfaces/cli/cli.go` — prints the shared build string through metadata-only early dispatch
- `runtime/internal/interfaces/http/server.go` — removed constructor-supplied version state and returns all build fields
- `runtime/internal/interfaces/http/events_test.go` and `runtime/tests/{proposal_detail,release_recovery}_test.go` — adapted call sites to the two-argument server constructor
- `studio/src/api/types.ts` and `studio/src/app/reachability.tsx` — carry commit and build date from system status
- `studio/src/features/shell/runtime-status.tsx` — visibly renders the Runtime version and exposes complete build metadata in its tooltip
- `studio/src/app/reachability-provider.test.tsx`, `studio/src/features/shell/shell.test.tsx`, and `studio/src/test/api-mock.ts` — updated status fixtures and rendering assertions
- `README.md`, `docs/ai/{tech-spec,repo-structure,design-system}.md`, `docs/ai/tickets/{GRF-223-build-metadata,INDEX}.md`, and this file — current build, API, image, UI, and completion contracts

**Files removed**

None.

**Contracts introduced or changed**

```go
// runtime/internal/buildinfo/buildinfo.go
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string

// runtime/internal/interfaces/cli/cli.go
func RunVersion(args []string, output io.Writer) (bool, error)

// runtime/internal/interfaces/http/server.go
func New(application *engine.Engine, logger *slog.Logger) *Server
```

```ts
// studio/src/api/types.ts
export type SystemStatus = {
	status: string;
	version: string;
	commit: string;
	buildDate: string;
	inference: string;
};
```

`GET /api/v1/system/status` now returns `version`, `commit`, and `buildDate`. The stable CLI/startup representation is `gyrifi {version} ({commit}, {date})`. Docker accepts `VERSION=dev`, `COMMIT=unknown`, and `BUILD_DATE=unknown`; it injects them with `-X` and publishes them through the matching OCI labels.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep three plain linker-set variables in a standard-library-only package | Every reporting surface can share deterministic values without I/O or package cycles | Derive the primary version from `runtime/debug.ReadBuildInfo()` | Working-tree Docker builds do not yield the required reliable metadata |
| Have HTTP read `buildinfo` directly | Build identity is process-wide, not server instance configuration | Preserve a constructor version argument | It permits another version source to drift and does not carry commit/date |
| Show only the concise version visibly and put full identity in the tooltip | Operators can identify the running release without crowding the status control | Render commit and timestamp inline | It would degrade the compact shell geometry |
| Mirror linker values into OCI labels | Binary output, HTTP, logs, and image inspection identify the same artifact | Labels without linker injection | The container metadata could disagree with the running process |

**Deviations from the ticket**

None. Every acceptance criterion was met.

**Traps for future work**

- Changing the HTTP constructor requires updating internal interface tests as well as Runtime integration tests; `events_test.go` was an internal call site.
- Version flags must run before `config.Load()` and data-directory creation; otherwise the default `/data` path can make a metadata-only command fail on a developer machine.
- Keep `buildinfo.String()` stable because scripts may parse it. New reporting fields do not justify changing that representation.
- Docker `ARG` values declared before the first `FROM` must be redeclared inside stages that consume them.
- The Studio must preserve metadata through `RuntimeHealth`; adding fields only to `SystemStatus` leaves the shell unable to render them.

**Tests added**

- `runtime/internal/buildinfo/buildinfo_test.go` — exact default and injected formatter output
- `runtime/internal/interfaces/cli/cli_test.go` — exact version output plus deferral of application-backed and unknown commands
- `runtime/tests/buildinfo_test.go` — HTTP status values are sourced from `buildinfo`
- `studio/src/features/shell/shell.test.tsx` — successful Runtime metadata produces visible version and the complete tooltip

**Docs updated**

- `docs/ai/tech-spec.md` §§2–3 and §13 — lifecycle, CLI, status response, and exact CI image assertion
- `docs/ai/repo-structure.md` — buildinfo package and image metadata contract
- `docs/ai/design-system.md` §3.3 — visible version and tooltip content
- `README.md` — versioned image build arguments and reporting surfaces
- `docs/ai/tickets/GRF-223-build-metadata.md` — acceptance results
- `docs/ai/tickets/INDEX.md` — GRF-223 marked Done
- `docs/ai/phases/phase-3.md` — this completion record

**Verification**

```text
$ cd runtime && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/cli
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 3.510s

$ go run ./cmd/gyrifi version
gyrifi dev (unknown, unknown)

$ cd studio && pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm coverage && pnpm build
Test Files  47 passed (47)
Tests       144 passed (144)
All files  | 86.15% Stmts | 85.69% Branch | 71.14% Funcs | 86.15% Lines
✓ 1867 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.29 kB
dist/assets/index-CXd8xUMU.css   42.89 kB │ gzip:  8.70 kB
dist/assets/index-DiYrk8HL.js   306.41 kB │ gzip: 92.02 kB
✓ built in 794ms

$ docker build --build-arg VERSION=9.9.9 --build-arg COMMIT=deadbeef --build-arg BUILD_DATE=2026-09-01T00:00:00Z -t gyrifi:grf-223 .
[+] Building 47.2s (31/31) FINISHED
IMAGE_VERSION=gyrifi 9.9.9 (deadbeef, 2026-09-01T00:00:00Z)
{"org.opencontainers.image.created":"2026-09-01T00:00:00Z","org.opencontainers.image.licenses":"AGPL-3.0-only","org.opencontainers.image.revision":"deadbeef","org.opencontainers.image.source":"https://github.com/Gyrifi-Ai/Gyrif-context-ledger","org.opencontainers.image.version":"9.9.9"}

$ ticket index consistency check
tickets consistent

$ git diff --check
(no output)

$ GitHub Actions branch-push run 33431187575
Runtime  completed  success
Studio   completed  success
Image    completed  success
```

**Follow-ups discovered**

None.

### GRF-224 — Health, readiness, and operational metrics

| | |
|---|---|
| Completed | 2026-08-31 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | Yes — the accepted-Change counter deliberately has no Ledger label |

**What was built**

The Runtime now exposes lock-free process liveness, bounded SQLite/migration readiness, asynchronously cached dependency health, and a standard-library Prometheus surface. Graceful shutdown marks the process unready before an optional drain delay and closes both the application and metrics listeners. Recovery-required Release Intents remain visible in status and metrics without making the only recovery API unready.

**Files added**

- `runtime/internal/config/config_test.go` — operational configuration defaults and invalid-bind/duration coverage
- `runtime/internal/engine/health.go` — readiness delegation and dependency/operational health snapshots
- `runtime/internal/engine/metrics.go` — domain metric sink and metered target adapter
- `runtime/internal/interfaces/http/health.go` — liveness, readiness, shutdown flag, and asynchronous health cache
- `runtime/internal/interfaces/http/health_test.go` — readiness failures, recovery anti-regression, gauges, and nonblocking cache tests
- `runtime/internal/interfaces/http/metrics.go` — atomic Prometheus collector and metrics-only handler
- `runtime/internal/interfaces/http/metrics_test.go` — format, bounded labels, domain outcome, route, and concurrency coverage

**Files changed**

- `.github/workflows/ci.yml` — image smoke test now waits for `/readyz` before asserting build identity
- `e2e/tests/harness.ts` — built-image readiness now requires `{"ready":true}` from `/readyz`
- `runtime/internal/bootstrap/bootstrap.go` — shared collector, dual listeners, unready/drain/shutdown order, and health-worker cleanup
- `runtime/internal/config/config.go` — loopback metrics address and non-negative drain delay configuration
- `runtime/internal/engine/{engine,changes,proposals,evaluation,releases,rollback}.go` — optional metric sink wiring and durable domain-outcome counters
- `runtime/internal/repository/{repository,sqlite,objects}.go` — exact-migration readiness and operational DB/object-store probes
- `runtime/internal/targets/{target,qdrant/qdrant}.go` — optional target health contract and Qdrant collection probe
- `runtime/internal/inference/{provider,llamacpp}.go` — optional provider health contract and llama.cpp health probe
- `runtime/internal/interfaces/http/server.go` — health/status routes, path-template request instrumentation, and operational-route SPA exclusions
- `studio/src/api/types.ts`, `studio/src/app/reachability.tsx`, `studio/src/app/reachability-provider.test.tsx`, and `studio/src/test/api-mock.ts` — enriched status health transport and fixtures
- `README.md`, `docs/ai/{product,repo-structure,tech-spec}.md`, `docs/ai/tickets/{GRF-224-health-and-metrics,GRF-232-e2e-suite,GRF-233-ci-pipeline,INDEX}.md`, and this file — operational contracts, closed gap, polling references, and completion bookkeeping

**Files removed**

None.

**Contracts introduced or changed**

```go
// runtime/internal/engine/engine.go
func New(repo repository.Repository, target targets.TargetAdapter, provider inference.Provider, sinks ...MetricSink) *Engine

// runtime/internal/engine/health.go
type SystemHealth struct {
	Database          string
	Target            string
	Inference         string
	UnresolvedIntents int64
	PendingChanges    int64
	ObjectStoreBytes  int64
}
func (engine *Engine) Readiness(ctx context.Context) (bool, error)
func (engine *Engine) ProbeHealth(ctx context.Context) SystemHealth

// runtime/internal/engine/metrics.go
type MetricSink interface {
	ChangeAccepted()
	ProposalCreated()
	EvaluationCompleted(bool)
	ReleaseCompleted(string)
	RollbackCreated()
	TargetRequest(operation, outcome string)
}

// runtime/internal/repository/repository.go additions
type OperationalStats struct {
	UnresolvedIntents int64
	PendingChanges    int64
}
Readiness(context.Context) (bool, error)
DatabaseStats(context.Context) (OperationalStats, error)
ObjectStoreBytes(context.Context) (int64, error)

// runtime/internal/targets/target.go and runtime/internal/inference/provider.go
type HealthChecker interface {
	Health(context.Context) error
}

// runtime/internal/interfaces/http/server.go and metrics.go
func New(application *engine.Engine, logger *slog.Logger, collectors ...*Metrics) *Server
func NewMetrics() *Metrics
func (server *Server) SetShuttingDown()
func (server *Server) MetricsHandler() http.Handler
```

`GET /healthz` returns text `ok`. `GET /readyz` returns `{"ready":true}` or HTTP 503 with `database_unreachable`, `migrations_incomplete`, or `shutting_down`. `GET /api/v1/system/status` adds `health.database`, `health.target`, `health.inference`, and `health.unresolvedIntents`. The separate `GYRIFI_METRICS_ADDRESS` listener serves only `GET /metrics` as Prometheus text 0.0.4; `GYRIFI_DRAIN_DELAY` controls the interval between becoming unready and listener shutdown.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep `/healthz` independent of Engine, repository, locks, and health caches | Liveness must answer even while SQLite is locked or broken | Reuse readiness or status | Dependency failure would cause an orchestrator to kill a live, diagnosable process |
| Readiness checks only SQLite plus the exact embedded migration names | These are the minimum conditions for every request to use the local ledger correctly | Include Qdrant, inference, or unresolved Intents | Those affect specific operations; draining the recovery API would worsen an incident |
| Refresh dependency and operational health asynchronously every 15 seconds | Status and metrics remain bounded when Qdrant or inference hangs | Probe synchronously on every scrape/request | External dependencies would control operator endpoint latency and duplicate probes |
| Bind metrics to a separate loopback-only listener, default `127.0.0.1:9090` | Operational data stays off the application/auth surface and is safe before GRF-220 | Serve `/metrics` on `:8080` | It would be unauthenticated network exposure now and require auth middleware later |
| Drop the Ledger label from accepted Changes | A deployment can create unbounded Ledgers, so even IDs are unsafe cardinality | Emit Ledger name or ID | Names may be sensitive and both names and IDs are unbounded |
| Count durable outcomes in Engine and meter target calls through an adapter | Counters reflect committed domain facts, including internal callers | Increment in HTTP handlers | Retries, idempotent reads, and non-HTTP callers would produce incorrect totals |

**Deviations from the ticket**

The counter list named `gyrifi_changes_accepted_total{ledger}`. The implemented metric is deliberately `gyrifi_changes_accepted_total` with no label because Ledger count is unbounded; this follows the ticket's later instruction to drop that label when deployments can have many Ledgers. No other acceptance criterion deviated.

**Traps for future work**

- Go method-aware `Request.Pattern` contains both method and route; strip the method before using it as `path_template`, otherwise the method is represented twice.
- The health cache owns background goroutines. Cancel and join them before closing the repository or race-enabled shutdown tests can observe probes against a closed database.
- Metrics must count durable outcomes, not method entry. Idempotent Change retries and failed rollback construction must not increment domain counters.
- `/metrics` must remain absent from the application listener; its explicit JSON 404 prevents the SPA fallback from returning healthy-looking HTML.
- `RECOVERY_REQUIRED` belongs in status and `gyrifi_unresolved_intents`, never readiness.

**Tests added**

- `runtime/internal/config/config_test.go` — loopback-only metrics binding, defaults, custom drain duration, and invalid configuration
- `runtime/internal/interfaces/http/health_test.go` — no-DB liveness, bounded DB and migration failures, shutdown, recovery-required readiness, exact gauges, and nonblocking cached dependency probes
- `runtime/internal/interfaces/http/metrics_test.go` — Prometheus headers/escaping, every route template, ID-cardinality exclusion, real success/failure/rollback flows, and concurrent race safety

**Docs updated**

- `docs/ai/product.md` §7 — removed the missing-observability gap
- `docs/ai/tech-spec.md` §§2–3, §6–§7, and §14 — lifecycle, configuration, endpoints, metrics, health contracts, E2E readiness, and closed gap
- `docs/ai/repo-structure.md` §2 — new config, Engine, and HTTP health/metrics files
- `README.md` §Quick start and §Configuration — probes, loopback scraping, and operational environment keys
- `docs/ai/tickets/GRF-224-health-and-metrics.md` — acceptance results
- `docs/ai/tickets/GRF-232-e2e-suite.md` and `GRF-233-ci-pipeline.md` — readiness polling now targets `/readyz`
- `docs/ai/tickets/INDEX.md` and this file — ticket completion status and implementation record

**Verification**

```
$ test -z "$(gofmt -l .)"
runtime format: clean
$ go vet ./...
runtime vet: passed
$ go test ./... -race
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/config 1.475s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine 1.978s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http 4.167s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository 3.503s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant 4.100s
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 6.107s
$ go build ./...
runtime build: passed

$ pnpm install --frozen-lockfile
Already up to date
Done in 196ms using pnpm v11.15.1
$ pnpm typecheck
$ pnpm test
Test Files  48 passed (48)
Tests  152 passed (152)
$ pnpm coverage
Test Files  48 passed (48)
Tests  152 passed (152)
All files | 86% Stmts | 84.4% Branch | 71.61% Funcs | 86% Lines
$ pnpm build
✓ 1868 modules transformed.
✓ built in 808ms

$ docker build -t gyrifi:local .
[+] Building 33.5s (31/31) FINISHED
=> => naming to docker.io/library/gyrifi:local

$ node node_modules/@playwright/test/cli.js test --list
Total: 2 tests in 1 file

$ ticket index consistency check
tickets consistent
$ git diff --check
diff whitespace: clean
```

**Follow-ups discovered**

GRF-220 must continue exempting `/healthz` and must not move metrics onto the application listener. GRF-226 must exempt `/healthz` from rate limiting. Both are already explicit acceptance criteria in their existing tickets; no new ticket was required.

### GRF-220 — Trusted deployment boundary decision

| | |
|---|---|
| Completed | 2026-09-01 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | Yes — application authentication was explicitly rejected by the owner |

**What was built**

ADR 0002 now defines Gyrifi as a local-first service deployed inside a company-controlled VM, private VPC, VPN, service mesh, firewall, or authenticated reverse proxy. The decision closes GRF-220 without adding application users, passwords, sessions, ingestion tokens, auth middleware, auth UI, a migration, or a dependency. Current product, technical, repository, and operator documentation now states that every admitted caller has full governance authority and public-internet exposure is unsupported.

**Files added**

- `docs/adr/0002-authentication-model.md` — accepted trusted-boundary architecture decision and revisit triggers

**Files changed**

- `README.md` — supported private-deployment boundary and caller-asserted approval identity
- `docs/ai/product.md` — trust model and removal of application authentication as a product gap
- `docs/ai/tech-spec.md` — trusted deployment contract and removal of the closed technical gap
- `docs/ai/repo-structure.md` — Compose/private-interface guidance
- `docs/ai/tickets/GRF-220-authentication.md` — superseded resolution while retaining rejected criteria as history
- `docs/ai/tickets/GRF-226-rate-limiting.md` — address-keyed limiting with no authentication dependency
- `docs/ai/tickets/GRF-211-proposal-detail-api.md` and `GRF-232-e2e-suite.md` — removed future application-auth assumptions
- `docs/ai/tickets/GRF-227-local-docker-launch.md` — corrected deployment-boundary acceptance wording
- `docs/ai/tickets/INDEX.md` — GRF-220 completion, GRF-226 dependency, and prior phase-log reference repairs
- `runtime/internal/interfaces/http/server.go` — removed the obsolete GRF-220 evidence-authorization TODO
- `docs/ai/phases/phase-3.md` — current phase contracts and this completion record

**Files removed**

None.

**Contracts introduced or changed**

There is deliberately no new Runtime wire, schema, config, CLI, Go, or TypeScript contract. The deployment contract is:

```text
application authentication: none
supported admission boundary: company VM / private VPC / VPN / service mesh / firewall / authenticated reverse proxy
public internet exposure: unsupported
approval actor: caller-asserted, not cryptographically verified
metrics: separate loopback-only listener
```

Migration `005` remains available to the next schema-changing ticket. GRF-226 now has no dependency on GRF-220 and keys all limited routes by validated client address.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Delegate admission and identity to company infrastructure | The owner specified a local-first system running inside a controlled company VM or VPC | Built-in passwords, sessions, and ingestion tokens | Duplicates company identity infrastructure and adds credential/session operations outside the desired product scope |
| Keep the lack of verified actor identity explicit | The audit record must not imply a guarantee the product does not provide | Describe `actor` as authenticated identity | No application credential binds the supplied value to a person |
| Keep local Compose loopback-only | It is the safest zero-configuration development boundary | Publish `8080` on all host interfaces | A default wildcard host bind would normalize unsafe accidental exposure |
| Retain GRF-226 with address-based keys | Trusted clients can still be buggy or compromised and starve the runtime | Drop rate limiting with auth | Availability protection remains valuable without application identity |

**Deviations from the ticket**

Every original implementation acceptance criterion for credential storage, bearer-token authentication, signed Operator sessions, route authorization, auth CLI operations, secret non-leakage tests, login UI, token management UI, and authenticated approval identity was intentionally not implemented. The owner explicitly decided that no application authentication is needed for the company VM/VPC deployment model and approved ADR 0002 on 2026-09-01. The accepted deliverable is the replacement trust-boundary decision and accurate documentation; the rejected criteria remain unchecked in the ticket so they cannot be mistaken for implemented behavior.

**Traps for future work**

- A private VPC is an admission boundary, not caller identity. Do not describe approval actors as verified.
- Public exposure remains unsupported. A future public or multi-tenant deployment requires a new ADR, not a silent bind-address change.
- Reverse-proxy forwarding headers are untrusted unless the immediate peer is configured as trusted; GRF-226 owns that enforcement.
- Historical phase entries mention future GRF-220 authentication because they record the decisions at that time. Current reference docs and ADR 0002 supersede those forward-looking notes.

**Tests added**

None — no executable behavior was added. The obsolete authorization TODO was removed, and the full existing Runtime/Studio/image gate verifies that the documentation-only scope decision did not disturb the product.

**Docs updated**

- `docs/adr/0002-authentication-model.md` — accepted trust model
- `docs/ai/product.md` §§1, 2, and 7 — deployment boundary, approval identity, and gap closure
- `docs/ai/tech-spec.md` §§1–2, §3, and §14 — distribution, trust boundary, metrics wording, and gap closure
- `docs/ai/repo-structure.md` §4 — local and private-interface deployment rules
- `README.md` §Quick start — operator-facing deployment warning
- `docs/ai/tickets/GRF-220-authentication.md` — superseded resolution
- `docs/ai/tickets/GRF-226-rate-limiting.md` — address-keyed replacement scope
- `docs/ai/tickets/INDEX.md` and this file — completion bookkeeping

**Verification**

```
$ test -z "$(gofmt -l .)"
runtime format: clean
$ go vet ./...
runtime vet: passed
$ go test ./... -race
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http 2.106s
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 3.504s
$ go build ./...
runtime build: passed

$ pnpm install --frozen-lockfile
Already up to date
Done in 232ms using pnpm v11.15.1
$ pnpm typecheck
$ pnpm test
Test Files  48 passed (48)
Tests  152 passed (152)
$ pnpm build
✓ 1868 modules transformed.
✓ built in 833ms

$ docker build -t gyrifi:local .
[+] Building 31.3s (31/31) FINISHED
=> => naming to docker.io/library/gyrifi:local

$ ticket index consistency check
tickets consistent
$ git diff --check
diff whitespace: clean
```

**Follow-ups discovered**

GRF-226 remains required for availability protection but is now independent and address-keyed. Application authentication may be reconsidered only when an ADR 0002 revisit trigger occurs; it is not an untracked implementation gap.

### GRF-225 — Inference process supervision

| | |
|---|---|
| Completed | 2026-09-01 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

The optional `llama-server` child now has a process-lifetime supervisor rather than a one-shot startup. It continuously drains bounded stdout/stderr into structured debug logs, retains final stderr diagnostics, detects and reaps unexpected exits, and restarts with bounded exponential backoff until a configurable failure cap is reached. Runtime status exposes the precise supervisor state, evaluation returns a stable 503 without recording infrastructure failure as evidence, and Studio displays that outage separately from an evaluation result.

**Files added**

- `runtime/internal/inference/supervisor_test.go` — controllable child-process harness covering restart, cap/reset, shutdown, diagnostics, and resource cleanup
- `runtime/tests/inference_availability_test.go` — public status/evaluation and no-evidence persistence integration contract

**Files changed**

- `runtime/internal/inference/llamacpp.go` — bounded stream capture, retained stderr, supervised restart lifecycle, state reporting, and race-safe stop
- `runtime/internal/inference/llamacpp_test.go` — bounded output and structured logging tests
- `runtime/internal/inference/provider.go` — optional process-state reporting interface
- `runtime/internal/config/config.go` and `config_test.go` — configurable positive restart cap with default validation
- `runtime/internal/bootstrap/bootstrap.go` — supplies the process logger and restart cap
- `runtime/internal/engine/engine.go`, `evaluation.go`, and `health.go` — inference readiness/state, stable unavailability response, and coarse health mapping
- `runtime/internal/interfaces/http/health.go` and `server.go` — immediate cached process state and `inferenceState` status serialization
- `studio/src/api/types.ts` — typed inference lifecycle state
- `studio/src/features/proposals/evidence-panel.tsx` and `proposal-detail.test.tsx` — distinct infrastructure-outage alert and coverage
- `studio/src/app/reachability-provider.test.tsx` and `test/api-mock.ts` — updated system-status fixtures
- `docs/ai/product.md`, `tech-spec.md`, `repo-structure.md`, and `design-system.md` — current product, lifecycle, tree, and UI contracts
- `docs/ai/tickets/GRF-225-inference-supervision.md` and `INDEX.md` — accepted criteria and completion bookkeeping
- `docs/ai/phases/phase-3.md` — Phase 3 status and this completion record

**Files removed**

None.

**Contracts introduced or changed**

```go
func StartLlamaServer(ctx context.Context, logger *slog.Logger, executable, modelPath string, port, maxRestarts int) (*LlamaServer, error)

type StateReporter interface {
	Healthy() bool
	State() string
}

func (server *LlamaServer) Healthy() bool
func (server *LlamaServer) State() string
func (server *LlamaServer) Stop() error

func (engine *Engine) InferenceReady() bool
func (engine *Engine) InferenceState() string
```

```text
GYRIFI_INFERENCE_MAX_RESTARTS=5
```

`GET /api/v1/system/status` now includes `health.inferenceState` as `ready | starting | restarting | failed | stopped | disabled`, while retaining the GRF-224 coarse `health.inference` values. An evaluation made in a non-ready state returns HTTP 503 / `UNAVAILABLE` with `Evaluation is unavailable: the inference process is not running.` and writes no CheckResult.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Add optional `StateReporter` beside `Provider` | Engine can consume local process state without forcing state semantics onto every inference provider | Add `Healthy` and `State` directly to `Provider` | The ticket explicitly leaves the provider evaluation contract unchanged and non-process providers need not have this lifecycle |
| Give exactly one goroutine ownership of `Cmd.Wait()` and publish one buffered exit result | Readiness, supervision, and shutdown can observe one reaped process without concurrent `Wait` calls | Let `Stop`, readiness, and supervisor each call `Wait` | `exec.Cmd.Wait` is single-owner and concurrent calls race or lose exit state |
| Keep coarse health and add precise `inferenceState` | Existing GRF-224 clients retain `ok|disabled|unhealthy` while operators can see lifecycle detail | Replace `health.inference` with process states | It would silently break the existing wire contract and Studio fixtures |
| Reset the restart-failure counter only after readiness succeeds | A recovered child should not inherit earlier failed-start debt | Reset immediately after `Start` | A process that starts but never becomes ready would evade the crash-loop cap |

**Deviations from the ticket**

None. Every acceptance criterion was implemented. GRF-224 had already landed, so its coarse inference health field was preserved and augmented with `inferenceState` rather than redefined.

**Traps for future work**

- `Cmd.Wait()` must continue to have one owner. Pipe readers drain to EOF before that owner reaps the child so final model-load diagnostics are retained.
- `Stop()` and context cancellation are explicit supervisor signals; never infer intentional shutdown from an exit code because SIGTERM and crashes can overlap numerically.
- The retained stderr ring is diagnostic context, not a log store. Child lines remain bounded and verbose output stays at debug level.
- Race-instrumented helper binaries start materially slower than ordinary test binaries; fake children must remain alive long enough for the readiness probe to observe them.

**Tests added**

- `runtime/internal/inference/supervisor_test.go` — unexpected restart and state transition, crash-loop cap, readiness counter reset, cancellation without restart, repeated-cycle goroutine/file-descriptor cleanup, and startup stderr propagation
- `runtime/internal/inference/llamacpp_test.go` — bounded long-line retention and structured debug forwarding
- `runtime/tests/inference_availability_test.go` — exact HTTP 503, `health.inferenceState`, provider preflight, and zero CheckResult rows
- `studio/src/features/proposals/proposal-detail.test.tsx` — infrastructure outage remains distinct while existing evaluation evidence stays visible

**Docs updated**

- `docs/ai/tech-spec.md` §§2, 3, 6, 10, 12, and 14 — config, status/evaluation contracts, signatures, lifecycle, tests, and gap closure
- `docs/ai/product.md` §7 — removed the closed inference-supervision gap
- `docs/ai/design-system.md` §5.3 — inference infrastructure alert behavior
- `docs/ai/repo-structure.md` §1 — new supervisor and integration test files
- `docs/ai/tickets/GRF-225-inference-supervision.md`, `INDEX.md`, and this file — completion records

**Verification**

```text
$ test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference 6.572s
ok github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http (cached)
ok github.com/gyrifi/gyrif-context-ledger/runtime/tests 3.403s

$ pnpm install --frozen-lockfile
Already up to date
Done in 195ms using pnpm v11.15.1
$ pnpm typecheck
$ pnpm test
Test Files  48 passed (48)
Tests  153 passed (153)
$ pnpm build
✓ 1868 modules transformed.
✓ built in 813ms

$ docker build -t gyrifi:local .
[+] Building 36.1s (31/31) FINISHED
=> [runtime-build 8/8] RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build ... 27.7s
=> => naming to docker.io/library/gyrifi:local

$ ticket index consistency check
tickets consistent
```

**Follow-ups discovered**

None. Multiple inference workers, provider fallback, and model provisioning remain intentionally out of scope and do not require backlog changes.

### GRF-221 — Asynchronous Change preparation and base fingerprint

| | |
|---|---|
| Completed | 2026-09-01 |
| Commit / PR | Uncommitted workspace change |
| Deviated from ticket | No |

**What was built**

New Changes now commit as `ACCEPTED` and are prepared by a bounded in-process worker after release recovery. The worker atomically claims eligible rows, reads and validates the target outside SQLite transactions, then owner-guards READY, INVALID, or retry updates. It captures observed fingerprints, identifies no-ops, reclaims expired leases, exposes exhausted retries as stalled, and publishes completion events; Studio renders each outcome.

**Files added**

- `runtime/migrations/006_change_preparation.sql` — preparation lease, retry, semantic failure, no-op, and queue schema
- `runtime/internal/engine/preparation.go` — worker lifecycle and target/retry orchestration
- `runtime/internal/ledger/preparation_outcome_test.go` — exhaustive pure outcome table
- `runtime/tests/change_preparation_test.go` — target outcomes, retries, shutdown, no-target, and 200-Change concurrency coverage

**Files changed**

- `runtime/internal/{bootstrap,config,engine,ledger,repository,targets}` — worker composition, settings, contracts, persistence, Qdrant validation, events, and lifecycle integration
- `runtime/tests` and `runtime/internal/interfaces/http/*_test.go` — existing flows now wait for durable asynchronous readiness
- `studio/src/api/types.ts` and `studio/src/features/changes/*` — preparation wire fields and outcome presentation
- `docs/ai/{product,tech-spec,repo-structure,design-system}.md` — current preparation contracts
- `docs/ai/tickets/{GRF-221-change-preparation,INDEX}.md` and this file — completion records

**Files removed**

None.

**Contracts introduced or changed**

```go
type PreparationOptions struct {
	BatchSize int
	Lease     time.Duration
}
func (engine *Engine) StartPreparation(context.Context, PreparationOptions) error
func (engine *Engine) StopPreparation()

type ChangePreparer interface {
	Prepare(context.Context, ledger.Change) (Value, error)
}

func (repository *SQLite) ClaimChangesForPreparation(context.Context, string, int, int, time.Time, time.Time) ([]ledger.Change, error)
func (repository *SQLite) CompleteChangePreparation(context.Context, repository.PreparationUpdate) (bool, error)
```

```text
GYRIFI_PREPARE_BATCH_SIZE=25
GYRIFI_PREPARE_LEASE=2m
```

`Change` responses add `invalidReason?`, `noop`, and `stalled`. Migration 006 adds `prepare_owner`, `prepare_claimed_at`, `prepare_attempts`, `prepare_after`, `invalid_reason`, and `noop`. Events add `change.ready` and `change.invalid`.

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| Keep the ten-attempt ceiling internal | The ticket authorizes configuration only for batch size and lease | Add another environment key | Expands the operational contract beyond scope |
| Make Qdrant implement optional `ChangePreparer` | Desired semantic validation must happen before READY while ordinary targets can retain `Read` | Call `Compile` | Qdrant compilation also performs collection network I/O unrelated to local desired validation |
| Owner- and status-guard every completion | Reclaim or withdrawal must make stale work harmless | Update by Change ID alone | A delayed worker could overwrite `WITHDRAWN` or another owner's result |
| Wait for generated rollback Changes before constructing their Proposal | Existing rollback remains one operation while the READY-only invariant becomes real | Return a transient conflict after creating rollback Changes | Would break the established rollback API contract and force caller retries |

**Deviations from the ticket**

None. Qdrant currently has no batch-read API, so the optional implementation note to batch where supported does not apply; preparation uses its existing individual read contract.

**Traps for future work**

- The metered target wrapper does not expose optional adapter interfaces; preparation checks the original adapter retained in `targetHealth` for `ChangePreparer` and otherwise uses the metered `Read` path.
- Target I/O must remain between claim commit and completion transaction. Never move adapter calls into either repository method.
- The worker's attempt ceiling and `scanChange` stalled projection must change together.
- Rollback synthesis now depends on the preparation worker being started by bootstrap before the endpoint is served.

**Tests added**

- `runtime/internal/ledger/preparation_outcome_test.go` — all PUT/DELETE present/absent/equal outcomes
- `runtime/internal/repository/sqlite_test.go` — expired lease reclaim and stale-owner rejection
- `runtime/tests/change_preparation_test.go` — observed fingerprints, no-op, semantic INVALID reason, retry without invalidation, no-target readiness, cancellable shutdown, and exactly-once reads for 200 concurrent Changes
- `studio/src/features/changes/changes-page.test.tsx` — Preparing, semantic reason, and no-op rendering

**Docs updated**

- `docs/ai/product.md` §§2, 3, 5, and 7 — real lifecycle and gap closure
- `docs/ai/tech-spec.md` §§2, 6–8, 12, and 14 — settings, worker, wire/schema, tests, and gap closure
- `docs/ai/repo-structure.md` — preparation worker location
- `docs/ai/design-system.md` §5.2 — preparation outcome presentation

**Verification**

```text
$ test -z "$(gofmt -l .)" && go vet ./... && go test ./... -race && go build ./...
ok runtime/internal/engine 1.647s
ok runtime/internal/interfaces/http 3.428s
ok runtime/internal/repository 3.700s
ok runtime/tests 12.034s

$ pnpm install --frozen-lockfile && pnpm typecheck && pnpm test && pnpm build
Already up to date
Test Files 48 passed (48)
Tests 158 passed (158)
✓ 1868 modules transformed.
✓ built in 792ms

$ docker build -t gyrifi:local .
[+] Building 36.0s (31/31) FINISHED
=> => naming to docker.io/library/gyrifi:local

$ ticket index consistency check
tickets consistent
$ git diff --check
diff whitespace: clean
```

**Follow-ups discovered**

Qdrant advertises batch capability for apply semantics but exposes no batch-read adapter contract. A future performance ticket may add batch reads if preparation throughput demonstrates that individual reads are limiting; no dependency or speculative interface was added here.
