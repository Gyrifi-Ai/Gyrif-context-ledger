# Gyrifi Context Ledger

Gyrifi is a local-first governance layer for mutable AI context. Applications submit desired-state **Changes**; people collect them into reviewable **Proposals**; evaluation evidence and approvals bind to the exact Proposal hash; an approved Proposal becomes an immutable **Release** after Gyrifi safely applies and verifies it against the target context store.

Gyrifi does not replace the target database. SQLite owns Gyrifi metadata and governance state, local content-addressed objects retain values and rollback material, and the Qdrant adapter owns target interaction.

## Run with Docker

The supported distribution is one Docker image with one user-facing port:

```sh
docker build -t gyrifi:dev .
docker run --rm \
	-p 8080:8080 \
	-v gyrifi-data:/data \
	gyrifi:dev
```

Open <http://localhost:8080>. The same Go HTTP server provides Studio at `/`, the versioned API at `/api/v1/...`, and server-sent events at `/events/v1`.

The image runs as a non-root user, exposes only port `8080`, runs SQLite migrations at startup, and stores persistent state below `/data`. Node.js and the Go toolchain are build-stage dependencies only; neither is a production application server.

## Architecture

```text
React + TypeScript Studio
					│
					│ HTTP / SSE on :8080
					▼
Go interfaces (HTTP and CLI)
					│
					▼
				Engine
			┌───┼──────────────┐
			▼   ▼              ▼
	 Ledger Repository  TargetAdapter     InferenceProvider
	 (pure)  │              │                    │
					 ▼              ▼                    ▼
			 SQLite +         Qdrant          llama-server HTTP
			 local CAS                         (optional Gemma GGUF)
```

The repository is a modular monolith:

- `studio/` — React, TypeScript, Vite, and reusable UI.
- `runtime/cmd/gyrifi/` — the single Go executable.
- `runtime/internal/ledger/` — deterministic model, hashes, and invariants; no I/O.
- `runtime/internal/engine/` — Change, Proposal, evaluation, release, recovery, and rollback behavior.
- `runtime/internal/repository/` — Gyrifi-owned SQLite state and local content-addressed objects.
- `runtime/internal/targets/qdrant/` — Qdrant-specific target operations.
- `runtime/internal/inference/` — typed evaluation boundary and llama.cpp provider.
- `runtime/internal/interfaces/` — protocol parsing and response mapping only.
- `runtime/internal/bootstrap/` — composition root and lifecycle.
- `runtime/migrations/` — ordered SQLite migrations embedded into the executable.

Only the Engine release path calls `TargetAdapter.Apply`. Evaluation produces evidence, never authority. Rollback reconstructs retained prior values into new Changes and a new Proposal, so released history always moves forward.

## Local development

Requirements:

- Go 1.24 or newer
- Node.js 24 or newer
- pnpm 11
- Docker only when building/running the production image

Install and build Studio:

```sh
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
```

Run Studio with Vite during development:

```sh
pnpm dev
```

Vite listens on `5173` and proxies `/api` and `/events` to the Go runtime at `127.0.0.1:8080`.

Run the Go runtime:

```sh
cd runtime
GYRIFI_DATA_DIR=../.gyrifi go run ./cmd/gyrifi
```

The source build serves a small fallback page unless production Studio assets have been copied into `runtime/internal/interfaces/http/static/`. Vite is the normal local frontend during development; the Docker build embeds the real production assets before compiling Go.

Go quality gates:

```sh
cd runtime
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/gyrifi
```

The CLI uses the same Engine as HTTP:

```sh
cd runtime
GYRIFI_DATA_DIR=../.gyrifi go run ./cmd/gyrifi doctor
```

## Qdrant

Qdrant is the first and only target adapter. Configure it with environment variables:

```sh
docker run --rm \
	-p 8080:8080 \
	-v gyrifi-data:/data \
	-e GYRIFI_QDRANT_URL=http://host.docker.internal:6333 \
	-e GYRIFI_QDRANT_COLLECTION=context \
	-e GYRIFI_QDRANT_API_KEY=... \
	gyrifi:dev
```

The API key is optional and is never included in structured request logs. For Qdrant `PUT` Changes, the desired JSON value is a complete Qdrant point object, including its `id`, vector, and payload. The logical unit is the point ID used for reads, deletes, conflict checks, and history. The initial adapter supports Qdrant collections with one unnamed vector configuration; named multi-vector collections are rejected explicitly rather than verified with the wrong distance metric.

Qdrant sparse in-place updates are reported as **recoverable**, not falsely advertised as globally atomic. Gyrifi captures current fingerprints and before-images, persists a Release Intent, applies, verifies, and only then advances ledger `HEAD`.

## Optional Gemma evaluation with llama.cpp

The image includes `llama-server`, but no model. Deterministic governance works with inference disabled, which is the default.

Mount a compatible Gemma GGUF and enable the provider explicitly:

```sh
docker run --rm \
	-p 8080:8080 \
	-v gyrifi-data:/data \
	-v /absolute/path/gemma.gguf:/models/gemma.gguf:ro \
	-e GYRIFI_EVALUATION_PROVIDER=llamacpp \
	-e GYRIFI_MODEL_PATH=/models/gemma.gguf \
	gyrifi:dev
```

The Go process validates the model path, starts `/opt/llama/llama-server` on loopback port `8081`, waits for readiness, communicates over its OpenAI-compatible HTTP endpoint, and terminates it during shutdown. Port `8081` is not exposed by the image.

Natural-language output is parsed into Gyrifi-owned typed evidence containing `passed`, `summary`, `findings`, and model identity. Invalid free-form output is rejected. Evidence is persisted against the exact Proposal hash and cannot directly approve or release anything.

## Configuration

Configuration is loaded once in the composition root and injected into concrete components.

| Variable | Default | Purpose |
|---|---|---|
| `GYRIFI_HTTP_ADDRESS` | `:8080` | Studio/API listen address |
| `GYRIFI_DATA_DIR` | `/data` | Persistent data root |
| `GYRIFI_SQLITE_PATH` | `/data/state.db` | SQLite database path |
| `GYRIFI_OBJECTS_PATH` | `/data/objects` | Content-addressed object root |
| `GYRIFI_QDRANT_URL` | `http://127.0.0.1:6333` | Qdrant base URL |
| `GYRIFI_QDRANT_COLLECTION` | `gyrifi` | Governed collection |
| `GYRIFI_QDRANT_API_KEY` | empty | Optional Qdrant credential |
| `GYRIFI_EVALUATION_PROVIDER` | `disabled` | `disabled` or `llamacpp` |
| `GYRIFI_MODEL_PATH` | empty | Mounted Gemma GGUF path |
| `GYRIFI_LLAMA_SERVER_PATH` | image: `/opt/llama/llama-server` | llama-server executable |
| `GYRIFI_LLAMA_SERVER_PORT` | `8081` | Internal loopback port |
| `GYRIFI_LOG_LEVEL` | `info` | `info` or `debug` |

SQLite starts in WAL mode with `synchronous=FULL`, foreign keys enabled, and a bounded busy timeout. Migrations execute automatically before HTTP starts.

## Core API flow

The maintained contract uses `/api/v1`:

```text
POST /api/v1/ledgers
POST /api/v1/ledgers/{ledgerId}/changes
POST /api/v1/ledgers/{ledgerId}/proposals
POST /api/v1/ledgers/{ledgerId}/proposals/{proposalId}/evaluation
POST /api/v1/ledgers/{ledgerId}/proposals/{proposalId}/approvals
POST /api/v1/ledgers/{ledgerId}/proposals/{proposalId}/release
POST /api/v1/ledgers/{ledgerId}/releases/{releaseId}/rollback
```

A successful Change response is sent only after the SQLite transaction commits. Reusing an idempotency key with the same logical request returns the original Change; reusing it for different desired state returns a conflict. Proposal creation claims every selected Change transactionally. Checks and approvals are valid only for the current Proposal hash.

## Persistence and backups

Back up the mounted `/data` volume using a volume-aware snapshot mechanism while Gyrifi is stopped, or use a future repository-aware backup command. The state layout is intentionally simple:

```text
/data/
├── state.db
├── state.db-wal
├── state.db-shm
└── objects/
		└── <sha256 shards>
```

Object writes use a temporary file, sync, and atomic rename. The initial implementation intentionally does not include packs, remote object storage, delta compression, or background repacking.

## License

See [LICENSE](LICENSE).
