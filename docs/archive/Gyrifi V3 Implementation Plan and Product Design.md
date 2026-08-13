> **SUPERSEDED — HISTORICAL ONLY.** Current documentation is [docs/ai/](../ai/README.md).
> The design described here was replaced by [docs/ai/design-system.md](../ai/design-system.md).
> Where this document disagrees with the code, the code wins. See [AGENTS.md](../../AGENTS.md).

# Gyrifi V3 Implementation Plan and Product Design

## Implemented product shape

Gyrifi is one browser application backed by one Go modular monolith and distributed as one Docker image.

```text
docker run -p 8080:8080 -v gyrifi-data:/data gyrifi
    │
    └── http://localhost:8080
      ├── Studio
      ├── /api/v1
      └── /events/v1
```

The Studio top-level areas are **Ledgers**, **Changes**, **Proposals**, and **Releases**. Evaluation and review remain Proposal workflows. Recovery and rollback remain Release workflows.

## Repository layout

```text
gyrifi/
├── studio/
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── app/
│       ├── api/
│       ├── ui/
│       ├── features/
│       │   ├── ledgers/
│       │   ├── changes/
│       │   ├── proposals/
│       │   └── releases/
│       └── test/
├── runtime/
│   ├── cmd/gyrifi/
│   ├── internal/
│   │   ├── bootstrap/
│   │   ├── config/
│   │   ├── interfaces/http/
│   │   ├── interfaces/cli/
│   │   ├── ledger/
│   │   ├── engine/
│   │   ├── repository/
│   │   ├── targets/qdrant/
│   │   └── inference/
│   ├── migrations/
│   ├── tests/
│   ├── go.mod
│   └── go.sum
├── docs/adr/
├── e2e/
├── Dockerfile
├── pnpm-workspace.yaml
├── pnpm-lock.yaml
└── README.md
```

There is one Go module and one frontend package. No shared-package extraction or build orchestrator is needed.

## Product workflow

### 1. Create a Ledger

Studio creates a stable Ledger ID and user-facing name. The Repository inserts the Ledger and an empty HEAD transactionally. The configured Qdrant collection remains the corpus authority.

### 2. Accept Changes

Applications or Studio submit desired state to:

```text
POST /api/v1/ledgers/{ledgerId}/changes
```

The Go runtime validates JSON, action, logical unit, and idempotency identity; writes immutable desired bytes; and commits the Change to SQLite. The API returns `202` only after commit. Target mutation is forbidden here.

### 3. Construct a Proposal

Studio selects Ready Changes and creates a Proposal. SQLite's unique Change membership constraint prevents the same Change from entering two active Proposals. The Proposal captures current HEAD and receives a deterministic hash.

### 4. Evaluate and approve

The Engine asks the target for preview metadata and executes deterministic checks. If configured, it asks `InferenceProvider` for structured natural-language evidence. Evidence stores the exact Proposal hash, preview fidelity, model identity, summary, and findings.

A local actor approves only after current passing evidence exists. An approval records the same Proposal hash.

### 5. Release

The Engine serializes release work, rejects moved HEAD, captures current Qdrant fingerprints and before-images, and persists a Release Intent. It then calls Qdrant Apply and Verify. Only successful verification allows SQLite to insert the Release, advance HEAD, mark the Proposal released, and mark selected Changes released.

### 6. Recover or rollback

Startup loads unfinished Release Intents. A target that verifies desired can be finalized; uncertain state is marked recovery-required.

Selecting an older Release for rollback creates a new Proposal from retained before-images. The ordinary review and release path produces a new child of current HEAD.

## API contract

Current maintained endpoints:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/system/status` | Runtime/inference health |
| GET | `/api/v1/adapters` | Qdrant capabilities |
| GET/POST | `/api/v1/ledgers` | List/create Ledgers |
| GET/POST | `/api/v1/ledgers/{id}/changes` | List/accept Changes |
| GET/POST | `/api/v1/ledgers/{id}/proposals` | List/create Proposals |
| POST | `/api/v1/ledgers/{id}/proposals/{proposal}/evaluation` | Persist exact-hash evidence |
| POST | `/api/v1/ledgers/{id}/proposals/{proposal}/approvals` | Approve exact Proposal hash |
| POST | `/api/v1/ledgers/{id}/proposals/{proposal}/release` | Execute safe release |
| GET | `/api/v1/ledgers/{id}/releases` | Immutable Release history |
| POST | `/api/v1/ledgers/{id}/releases/{release}/rollback` | Create forward rollback Proposal |
| GET | `/events/v1` | Studio event stream/keepalive |

Errors have one protocol shape:

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Current passing evidence and approval are required."
  }
}
```

Raw SQLite, Qdrant, and llama-server errors are retained as wrapped log causes but not returned directly to users.

## Studio design

Studio is a frontend only. Vite provides local development and emits static production assets.

- `app/` owns route, providers, bootstrap, and shell.
- `api/` owns contract types, fetch client, and events.
- `ui/` contains reusable visual primitives, patterns, layout, and feedback only.
- `features/` owns all domain-aware UI.

The main shell is intentionally governance-oriented rather than infrastructure-oriented. There are no top-level pages for SQLite, object storage, target operations, inference processes, or Release Intents.

## Runtime composition

Bootstrap performs all construction:

```text
load configuration
  ↓
initialize /data
  ↓
open SQLite and set durability pragmas
  ↓
run embedded migrations
  ↓
construct local ObjectStore and Repository
  ↓
construct QdrantAdapter
  ↓
optionally start llama-server and construct LlamaCppProvider
  ↓
construct Engine
  ↓
construct CLI or HTTP interface
  ↓
recover unfinished Releases
  ↓
start :8080
```

Dependencies are constructor-injected. There is no global service locator.

## SQLite schema ownership

The initial migration creates:

```text
schema_migrations
ledgers
ledger_heads
changes
proposals
proposal_changes
checks
approvals
release_intents
releases
```

Important constraints include unique Ledger names, unique per-Ledger Change sequences, unique per-Ledger idempotency keys, and unique Change claims. Release finalization updates Release, HEAD, Proposal, Change, and Intent state in one SQLite transaction.

## Qdrant representation

The first adapter uses Qdrant's REST API and keeps Qdrant details inside `targets/qdrant`.

- Logical unit: point ID.
- `PUT` desired state: complete point JSON accepted by Qdrant's points upsert API.
- `DELETE`: point ID removal.
- Preview: declared `FAST` overlay fidelity.
- Apply: conditional fingerprint check followed by sparse point operation.
- Verify: read and compare canonical desired fingerprint or absence.
- Capability: recoverable sparse apply, not universal atomic visibility.

No placeholder adapters are included.

## Optional local evaluation

Local evaluation is off by default. With `GYRIFI_EVALUATION_PROVIDER=llamacpp`, bootstrap requires a configured GGUF, starts llama-server on loopback, performs health checks, and injects `LlamaCppProvider`.

The provider prompts for strict JSON and rejects unparsable free-form output. The resulting evidence is passive. Proposal policy in the Engine decides whether evidence passes; the provider has no Repository or TargetAdapter access.

## Docker design

The Dockerfile uses build stages:

1. Node + pnpm builds Studio.
2. The official llama.cpp server image supplies native runtime artifacts.
3. Go downloads modules, receives Studio output, runs Go tests, and builds one executable.
4. Ubuntu 24.04 provides an ABI-compatible minimal runtime with certificates and llama native libraries.

The final process is Go, running as non-root. `/data` is a volume. Only `8080` is exposed. llama-server uses internal loopback `8081` and has no exposed port.

## Test strategy

The implemented checks prioritize behavior:

- Deterministic, order-sensitive Proposal hashes.
- Approval binding to exact Proposal hash.
- Idempotent Change acceptance and conflicting key reuse.
- Change → Proposal → Evaluation → Approval → Release.
- Target apply failure does not advance Release history.
- Release → rollback Proposal → forward Release.
- Structured llama-server output parsing and free-form rejection.
- Qdrant collection path and API-key handling.
- Studio API contract and error mapping.
- SQLite migrations and persistence through integration flow.

Large Gemma downloads are never part of normal tests.

## Dependency rules

1. Studio UI has no governance rules.
2. HTTP and CLI call the same Engine.
3. Ledger performs no I/O.
4. Engine contains no SQL or filesystem implementation.
5. Engine has no Qdrant or llama.cpp protocol types.
6. Repository owns only Gyrifi state.
7. Qdrant owns only target interaction.
8. Inference produces evidence only.
9. Only release mutates the target.
10. Rollback creates forward history.
11. Configuration is loaded once.
12. No speculative adapters, providers, MCP packages, services, queues, or generic frameworks are added.
