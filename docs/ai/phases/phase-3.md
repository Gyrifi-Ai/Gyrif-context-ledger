# Phase 3 — Production hardening

**Goal:** make Gyrifi safe to run somewhere other than a developer's laptop. Everything here is a property the product currently lacks that would make a real deployment irresponsible.

**Status:** In progress

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-223](../tickets/GRF-223-build-metadata.md) | Build metadata and version consistency | S | — | Done |
| [GRF-221](../tickets/GRF-221-change-preparation.md) | Asynchronous Change preparation and base fingerprint | L | — | Not started |
| [GRF-222](../tickets/GRF-222-retention-backup.md) | Retention budgets, quotas, and backup command | L | — | Not started |
| [GRF-224](../tickets/GRF-224-health-and-metrics.md) | Health, readiness, and operational metrics | M | — | Not started |
| [GRF-225](../tickets/GRF-225-inference-supervision.md) | Inference process supervision | M | — | Not started |
| [GRF-220](../tickets/GRF-220-authentication.md) | Ingestion tokens and browser session auth | XL | — | Not started |
| [GRF-226](../tickets/GRF-226-rate-limiting.md) | Request rate limiting and abuse controls | M | GRF-220 | Not started |
| [GRF-227](../tickets/GRF-227-local-docker-launch.md) | Local Docker launch | M | — | Done |

Listed in recommended execution order, smallest and least entangled first.

## Phase-level notes

- **GRF-220 requires an ADR before implementation.** `docs/adr/0002-authentication-model.md` must be written and reviewed first. It changes the product's trust model, and that decision should outlive the ticket.
- **GRF-220 is the gate on any non-loopback deployment.** Until it lands, the only defensible deployment is `127.0.0.1`. Say so in any deployment guidance written before then.
- GRF-221 makes the `ACCEPTED` and `INVALID` change statuses real for the first time. Phase 1's Changes inbox will need updating to render them — that update is inside GRF-221's scope, not a Phase 1 regression.
- Migration numbers continue from Phase 2: 005 (GRF-220), 006 (GRF-221). Adjust and record here if the order changes.
- GRF-220 permits exactly one new dependency (`golang.org/x/crypto`). No other ticket in this phase adds one.

## The theme

Each ticket in this phase closes a way the product can fail silently:

| Ticket | Silent failure it prevents |
|---|---|
| GRF-223 | A bug report that cannot be tied to a build |
| GRF-221 | A Change whose relationship to the target's actual state is unknown |
| GRF-222 | A volume filling up mid-release, taking rollback material with it |
| GRF-224 | An orchestrator that cannot tell a healthy runtime from a broken one |
| GRF-225 | An inference process that died hours ago and took its logs with it |
| GRF-220 | An audit trail anyone on the network can forge |
| GRF-226 | One runaway client starving every operator out of the system |

## Exit criteria

- [ ] All seven tickets complete.
- [ ] The runtime reports one accurate, build-injected version everywhere.
- [ ] Every Change has a known relationship to the target's observed state before it can be proposed.
- [ ] Disk growth is bounded and a supported backup/restore procedure exists and is tested.
- [ ] No governance operation is reachable without an authenticated principal, and ingestion credentials cannot approve or release.
- [ ] Liveness, readiness, and metrics are exposed, and a `RECOVERY_REQUIRED` intent is visible without making the runtime unready.
- [ ] The inference child process is supervised, its output is captured, and its failures are legible.
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
