# Gyrifi Context Ledger

**Version control for the context your AI systems depend on.**

Gyrifi is a local-first governance layer for mutable AI context. Applications submit desired-state **Changes**. A person groups them into a reviewable **Proposal**. Evaluation evidence and approvals bind to the exact Proposal hash. Only then does Gyrifi apply the batch to the target vector store, verify it, and record an immutable **Release**.

Gyrifi does not replace your vector database. It sits in front of it and makes every mutation reviewable, reversible, and auditable.

```text
Application ──► Change ──► Proposal ──► Evaluation ──► Approval ──► Release ──► Qdrant
              durable inbox   batch      evidence      authority    verified
```

---

## Why

Vector stores are usually written to blindly. An embedding job runs, points are upserted, and nobody can answer:

- What changed in the knowledge base last Tuesday?
- Who approved this content?
- Can we get back to the state from before the bad ingestion run?

Gyrifi answers all three. Every mutation is a durable record, every batch is reviewed against retained before-images, and rollback is a first-class forward operation rather than a database restore.

---

## Core concepts

| Concept | Meaning |
|---|---|
| **Ledger** | A governed namespace. Owns its own Change inbox, Proposals, Releases, and `HEAD`. |
| **Change** | One desired-state mutation (`PUT` or `DELETE`) to one logical unit — for Qdrant, a point ID. Idempotent. |
| **Proposal** | An explicit, ordered selection of Ready Changes. Identified by a deterministic hash over ledger + base HEAD + ordered Change IDs. |
| **Evaluation** | Deterministic checks plus optional local LLM evidence. Produces evidence, never authority. |
| **Approval** | A human decision bound to an exact Proposal hash. Re-hashing a Proposal invalidates it. |
| **Release** | An immutable, parent-linked record written only after the target applied *and* verified. Advances `HEAD`. |
| **Rollback** | Reconstructs an earlier state from retained before-images into new Changes and a new Proposal. History only moves forward. |

---

## Quick start

Start the complete local system with one command:

```sh
docker compose up --build
```

Or choose **Gyrifi: Start local stack** from VS Code's **Run and Debug** view and press the Run button. This starts the compiled Studio and Go runtime in the `gyrifi` container, plus its local Qdrant target and a one-time collection initializer. No separate Go, Node, Studio, or Qdrant setup is needed.

Open <http://127.0.0.1:8080>. The same Go server serves Studio at `/`, the API at `/api/v1/...`, and the event stream at `/events/v1`. The stack creates the `gyrifi` Qdrant collection as a 3-dimensional cosine collection only when it does not already exist.

Gyrifi and Qdrant data persist in named Docker volumes. Stop the stack with `docker compose down`. To delete all local governance and target data, use `docker compose down --volumes`.

The Compose port is bound to `127.0.0.1` only. Until GRF-220 adds authentication, this local Compose configuration is not a production deployment mechanism.

The application image runs as a non-root user, exposes only `8080`, runs SQLite migrations at startup, and persists Gyrifi state under `/data`.

---

## Using Gyrifi

### 1. Create a Ledger

Use **Ledgers** in Studio, or:

```sh
curl -X POST localhost:8080/api/v1/ledgers \
  -H 'content-type: application/json' \
  -d '{"name":"product-docs","description":"Support knowledge base"}'
```

### 2. Submit Changes

For Qdrant the logical `unit` is the point ID and `desired` is the complete point object.

```sh
curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/changes \
  -H 'content-type: application/json' \
  -d '{
    "unit": "42",
    "action": "PUT",
    "desired": { "id": 42, "vector": [0.1, 0.2, 0.3], "payload": { "title": "Refunds" } },
    "idempotencyKey": "ingest-2026-08-12-42"
  }'
```

Returns `202 Accepted` **only after the SQLite transaction commits**. No target write happens here. Retrying with the same key and the same body returns the original Change; the same key with a different body is a `409`.

### 3. Build a Proposal

```sh
curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/proposals \
  -H 'content-type: application/json' \
  -d '{"title":"August refund policy refresh","changeIds":["chg_...","chg_..."]}'
```

Every selected Change is claimed transactionally. A Change belongs to at most one Proposal.

### 4. Evaluate and approve

```sh
curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/proposals/$PR/evaluation \
  -H 'content-type: application/json' \
  -d '{"criteria":"No PII; policy statements must cite a source."}'

curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/proposals/$PR/approvals \
  -H 'content-type: application/json' \
  -d '{"actor":"you@example.com"}'
```

Approval is refused unless a **current passing** evaluation exists for the Proposal's exact hash.

### 5. Release

```sh
curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/proposals/$PR/release
```

Gyrifi validates `HEAD`, compiles the plan, captures before-images, persists a Release Intent, applies to Qdrant, verifies, and only then inserts the Release and advances `HEAD` in one SQLite transaction.

### 6. Roll back

```sh
curl -X POST localhost:8080/api/v1/ledgers/$LEDGER/releases/$OLD_RELEASE/rollback
```

This does **not** rewind. It creates a new Proposal whose Changes restore the state as of the selected Release, then follows the ordinary evaluation → approval → release path.

---

## Configuration

Configuration is loaded once at startup and injected. No config file, no service discovery.

| Variable | Default | Purpose |
|---|---|---|
| `GYRIFI_HTTP_ADDRESS` | `:8080` | Studio/API listen address |
| `GYRIFI_DATA_DIR` | `/data` | Persistent data root |
| `GYRIFI_SQLITE_PATH` | `$GYRIFI_DATA_DIR/state.db` | SQLite database path |
| `GYRIFI_OBJECTS_PATH` | `$GYRIFI_DATA_DIR/objects` | Content-addressed object root |
| `GYRIFI_QDRANT_URL` | `http://127.0.0.1:6333` | Qdrant base URL |
| `GYRIFI_QDRANT_COLLECTION` | `gyrifi` | Governed collection |
| `GYRIFI_QDRANT_API_KEY` | empty | Optional Qdrant credential, sent as `api-key` |
| `GYRIFI_EVALUATION_PROVIDER` | `disabled` | `disabled` or `llamacpp` |
| `GYRIFI_MODEL_PATH` | empty | GGUF path; required when provider is `llamacpp` |
| `GYRIFI_LLAMA_SERVER_PATH` | `llama-server` (image: `/opt/llama/llama-server`) | llama-server executable |
| `GYRIFI_LLAMA_SERVER_PORT` | `8081` | Internal loopback port, never exposed |
| `GYRIFI_LOG_LEVEL` | `info` | `info` or `debug` |

SQLite runs with `journal_mode=WAL`, `synchronous=FULL`, `foreign_keys=ON`, and `busy_timeout=5000`. Migrations run automatically before HTTP starts.

---

## Optional local evaluation

The image ships `llama-server` but no model. Deterministic governance works with inference disabled, which is the default.

```sh
docker run --rm -p 8080:8080 \
  -v gyrifi-data:/data \
  -v /abs/path/gemma.gguf:/models/gemma.gguf:ro \
  -e GYRIFI_EVALUATION_PROVIDER=llamacpp \
  -e GYRIFI_MODEL_PATH=/models/gemma.gguf \
  gyrifi:dev
```

Go supervises `llama-server` on loopback `8081`, waits for `/health`, and terminates it on shutdown. Model output is parsed into typed evidence (`passed`, `summary`, `findings`, model identity). Free-form output is rejected. Evidence can never approve or release on its own.

---

## Local development

Requirements: Go 1.24+, Node.js 24+, pnpm 11. Docker only for the image.

```sh
# Studio
pnpm install --frozen-lockfile
pnpm typecheck && pnpm test && pnpm build
pnpm dev            # Vite on :5173, proxies /api and /events to :8080

# Runtime
cd runtime
GYRIFI_DATA_DIR=../.gyrifi go run ./cmd/gyrifi
go fmt ./... && go vet ./... && go test ./... && go build ./cmd/gyrifi
```

The source build serves a fallback page unless production Studio assets are copied into `runtime/internal/interfaces/http/static/`. The Docker build does this automatically.

### Browser end-to-end qualification

Requirements: Docker with Compose and the Node.js/pnpm versions above. The e2e package is intentionally separate from the Studio workspace and builds the current repository image before every suite:

```sh
cd e2e
pnpm install --ignore-workspace --frozen-lockfile
pnpm --ignore-workspace install:browser
pnpm --ignore-workspace test
```

The suite uses loopback ports `18082` and `16333`, creates and removes dedicated named volumes, and runs Chromium only. Override the ports with `GYRIFI_E2E_PORT` and `GYRIFI_E2E_QDRANT_PORT` if needed. To repeat every isolated journey three times for stability qualification, run `pnpm --ignore-workspace test:repeat`. Failure traces and screenshots are written under `e2e/test-results/`.

The CLI uses the same Engine as HTTP:

```sh
go run ./cmd/gyrifi doctor     # ledger count and inference state as JSON
go run ./cmd/gyrifi version
```

---

## Persistence and backups

```text
/data/
├── state.db          # governance state
├── state.db-wal
├── state.db-shm
└── objects/          # sha256-addressed values and before-images
    └── ab/cdef…
```

Back up the whole `/data` volume with Gyrifi stopped. Object writes use a temp file, `fsync`, and atomic rename. There are deliberately no packs, remote object stores, deltas, or repacking.

---

## Architecture at a glance

```text
React + TypeScript Studio
      │ HTTP / SSE :8080
      ▼
Interfaces (HTTP, CLI)
      ▼
   Engine ──► Ledger (pure, no I/O)
      ├────► Repository        → SQLite (WAL) + local CAS
      ├────► TargetAdapter     → Qdrant REST
      └────► InferenceProvider → llama-server HTTP (optional)
```

One Go module, one frontend package, one image. No Rust, no Python runtime, no production Node server, no microservices.

---

## Documentation

Deep documentation lives in [docs/ai/](docs/ai/README.md) and is written to be read one file at a time:

| Document | Read it when |
|---|---|
| [docs/ai/product.md](docs/ai/product.md) | You need the domain model, workflows, and invariants |
| [docs/ai/repo-structure.md](docs/ai/repo-structure.md) | You need to know where code goes |
| [docs/ai/tech-spec.md](docs/ai/tech-spec.md) | You need API shapes, schema, types, algorithms |
| [docs/ai/design-system.md](docs/ai/design-system.md) | You are touching Studio UI |
| [docs/ai/tickets/INDEX.md](docs/ai/tickets/INDEX.md) | You are picking up work |
| [docs/ai/phases/](docs/ai/phases/README.md) | You want the history of what was built and why |
| [AGENTS.md](AGENTS.md) | You are an AI agent working in this repo |

Architecture decisions live in [docs/adr/](docs/adr/). Superseded design documents are kept in [docs/archive/](docs/archive/) for history only — **they do not describe the current system.**

---

## License

See [LICENSE](LICENSE).
