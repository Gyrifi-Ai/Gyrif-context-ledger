> **SUPERSEDED — HISTORICAL ONLY.** Current documentation is [docs/ai/](../ai/README.md).
> Where this document disagrees with the code, the code wins. See [AGENTS.md](../../AGENTS.md).

# Gyrifi V3: Local-First Context Ledger Standard

## Status and technology baseline

This document defines the implemented V3 architecture. Gyrifi is a modular monolith distributed as one Docker image. The brand is **Gyrifi**; the executable and system are **gyrifi**.

The settled stack is:

| Area | Decision |
|---|---|
| User application | React + TypeScript Studio |
| Frontend build | Vite + pnpm |
| Runtime | Go modular monolith |
| Gyrifi state | SQLite in WAL mode |
| Large immutable values | Local SHA-256 content-addressed objects |
| First target | Qdrant |
| Optional local evaluation | llama.cpp `llama-server` over loopback HTTP |
| Initial local SLM | Gemma GGUF supplied by the user |
| Distribution | One multi-stage Docker image, one public port |

There is no Rust or Python runtime, no production Node.js server, and no microservice control plane. Node/pnpm and Go are build-stage tools. Optional local inference may use a child process, but remains inside the one Gyrifi container and behind the one Gyrifi product interface.

## Product boundary

Gyrifi governs proposed mutations to a context store. It does not become the target database and does not own the full live corpus.

```text
Actual context corpus                 → Qdrant
Governance metadata and current HEAD → SQLite
Proposed/rollback values             → SQLite + local immutable objects
Evaluation and approval evidence     → SQLite
Target mutation                      → Release path only
```

The user-visible model remains deliberately small:

```text
Ledger
  ├── Change
  ├── Proposal (Context PR)
  └── Release
```

A **Change** is one desired-state mutation to one logical unit. A **Proposal** is an explicit ordered selection of Changes, review evidence, and approvals. A **Release** is an immutable record created only after target apply and verification. `HEAD` points to the newest finalized Release.

## Runtime dependency structure

```text
Studio
   │ HTTP / SSE
   ▼
Interfaces
   │
   ▼
Engine
   ├────────► Ledger
   ├────────► Repository
   ├────────► TargetAdapter
   └────────► InferenceProvider

Repository        = SQLite + local objects
TargetAdapter     = Qdrant
InferenceProvider = llama-server HTTP (optional)
```

### Ledger

The Ledger package answers “what is valid?” It owns IDs, canonical hashes, logical-unit identity, state values, and invariants. It is deterministic and has no external I/O. It must not import or describe HTTP, CLI, SQLite, filesystems, Qdrant, llama.cpp, environment variables, or React.

### Engine

The Engine is the application facade. HTTP and CLI call the same instance. Its areas are files and focused methods rather than separately deployable services:

- Change acceptance and idempotency.
- Proposal selection, exact hash identity, checks, approvals, and gates.
- Evaluation against an effective proposed state.
- Release Intent, target apply, verification, finalization, restart recovery, and rollback planning.

Only release methods may call target mutation operations.

### Repository

Repository methods describe Gyrifi operations, not generic persistence. SQLite owns ledgers, Changes, Proposal claims, checks, approvals, Release Intents, Releases, and HEAD. The local object store owns immutable values and before-images.

SQLite is configured with:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

Ordered migrations execute automatically before the HTTP server starts. A successful ingestion response is emitted only after the Change transaction commits.

### TargetAdapter

TargetAdapter owns external target semantics:

```text
Read
Fingerprint
Preview
Compile
Apply
Verify
Restore
Capabilities
```

The Engine contains no Qdrant request types. Qdrant is the only current adapter. Its sparse in-place release guarantee is **recoverable** rather than universally atomic.

### InferenceProvider

InferenceProvider accepts a Gyrifi-owned typed request and returns typed evidence. The initial provider calls the OpenAI-compatible llama-server HTTP API. The Engine does not know model flags, GGUF internals, Gemma API details, or process arguments.

Model output cannot approve or release. It is evidence associated with exactly one Proposal hash.

## Required invariants

1. An acknowledged Change is durably represented in SQLite.
2. One `(ledger, idempotency key)` maps to one logical request forever.
3. A Change belongs to one Ledger and addresses one stable logical unit.
4. A Ready Change can be claimed by at most one active Proposal.
5. Proposal membership is explicit and ordered.
6. Proposal identity includes Ledger, base HEAD, and ordered Change IDs.
7. Checks and approvals match the current Proposal hash or are stale.
8. Evaluation never mutates the target and never authorizes release by itself.
9. Release checks current HEAD before applying.
10. A persisted Release Intent and retained before-images exist before target mutation.
11. `HEAD` advances only after target verification succeeds.
12. Unfinished external application is recovered or marked recovery-required; it is not guessed successful.
13. Only the Release path may call target Apply/Restore.
14. Releases are immutable and parent-linked.
15. Rollback reconstructs desired state into new Changes and creates a new forward Proposal/Release.
16. Physical object representation never changes logical hashes.
17. Local inference can be disabled without disabling deterministic governance.

## Change acceptance

The Change API is versioned:

```text
POST /api/v1/ledgers/{ledgerId}/changes
```

A request includes a logical unit, `PUT` or `DELETE`, desired JSON for `PUT`, and an idempotency key. Gyrifi canonicalizes and fingerprints the request, writes an immutable value object, inserts the complete hot Change row, and commits before returning `202 Accepted`.

No target write occurs during acceptance. If the same key and request are retried, the existing Change is returned. The same key with different contents is a conflict.

## Proposal, evaluation, and approval

Proposal creation atomically checks every selected Change and inserts unique claims. A partial Proposal is not created when one Change is unavailable.

The canonical Proposal hash is deterministic and order-sensitive. Evaluation records preview fidelity and either deterministic evidence or structured model evidence. A passing current evaluation is required before approval. Approval records actor and current Proposal hash. Release readiness is computed by the Engine, never trusted from Studio state.

Natural-language evaluation is optional. When disabled, deterministic checks and the Qdrant preview contract continue to work.

## Release and recovery

Release is the crash-sensitive boundary between SQLite and Qdrant:

```text
lock release path
  ↓
load Proposal + current evidence + approval
  ↓
validate Proposal base against HEAD
  ↓
compile target plan
  ↓
read current target units
  ↓
capture fingerprints and immutable before-images
  ↓
persist Release Intent
  ↓
apply Qdrant operations
  ↓
verify desired state
  ↓
insert immutable Release + advance HEAD in one SQLite transaction
```

A failure before target apply leaves HEAD unchanged. A target apply or verification failure leaves a durable unfinished/recovery-required Intent. At startup, Gyrifi inspects unfinished Intents and finalizes only target state that verifies as desired.

## Rollback

Rollback never moves HEAD backward. To restore an older Release, Gyrifi walks newer Release plans backward, resolves retained before-image objects, reduces them by logical unit, and creates ordinary rollback Changes. Those Changes form a new Proposal based on current HEAD and pass through evaluation, approval, Release Intent, apply, and verification.

This produces:

```text
R1 → R2 → R3 → R4
               └── R4 restores state represented by R1
```

If required rollback material has expired or is corrupt, rollback is blocked rather than fabricated.

## Local object storage

Objects are addressed by SHA-256 over type and content. Writes use a same-directory temporary file, restrictive permissions, sync, and atomic rename. Existing hashes are reused.

V3 intentionally does not implement pack files, remote object storage, deltas, multi-pack indexes, or background repacking. Those mechanisms are not required for the current product scale.

## Docker and process lifecycle

The image serves one public port (`8080`) and persists `/data`. Studio, API, and events share the Go server. llama-server binds to `127.0.0.1:8081` only when enabled and is not exposed.

Go is the parent process and handles SIGTERM/SIGINT. It gracefully shuts down HTTP, cancels in-flight work, terminates llama-server, closes SQLite, and exits. No Supervisor, Compose requirement, systemd, Kubernetes, message queue, or event bus is involved.

## Explicit non-goals

- Multiple Gyrifi replicas sharing one repository.
- Microservices or generic workflow engines.
- PostgreSQL/Redis/Kafka for Gyrifi state.
- Plugin marketplaces or adapter SDKs.
- Placeholder target adapters.
- Python inference services or direct CGO llama bindings.
- Automatic model downloads in tests or startup.
- Full target-database snapshots.
- Remote object archives and speculative scale infrastructure.
