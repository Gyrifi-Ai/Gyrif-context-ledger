> **SUPERSEDED — HISTORICAL ONLY. DO NOT IMPLEMENT FROM THIS FILE.**
>
> The `GRF-1XX` ticket IDs below are **retired** and have no ticket files. The current
> backlog is [docs/ai/tickets/INDEX.md](../ai/tickets/INDEX.md) using `GRF-2XX` IDs.
> Where this document disagrees with the code, the code wins. See [AGENTS.md](../../AGENTS.md).

# Gyrif V3 Implementation Roadmap

## Delivery contract

Gyrifi V3 is a Go + React modular monolith distributed as one Docker image. The basic user journey is:

```text
docker run -p 8080:8080 -v gyrifi-data:/data gyrifi
  ↓
Studio
  ↓
Ledger → Change → Proposal → Evaluation → Approval → Release
  ↓
forward rollback when needed
```

The runtime owns SQLite governance state and local content-addressed objects. Qdrant is the first target. Optional Gemma GGUF evaluation runs through a supervised loopback llama-server child process. No Rust, Python runtime, production Node server, or microservices belong in V3.

## Implemented foundation

### GRF-101 — Repository and build structure

**Outcome:** pnpm workspace, React/Vite Studio, one Go module, modular internal packages, migrations, tests, Dockerfile, and ADR.

**Validation:** Studio and Go build independently; the final image has one exposed port and a non-root user.

### GRF-102 — Deterministic Ledger model

**Outcome:** I/O-free Ledger, Change, Proposal, Release, HEAD, hash, fingerprint, approval, and state types.

**Rules:** Ledger imports no HTTP, SQLite, filesystem, Qdrant, inference, or environment concerns. Proposal hashes are deterministic and order-sensitive. Release identity includes parent and Proposal hash.

### GRF-103 — SQLite Repository

**Outcome:** automatic embedded migrations and operation-specific Repository methods.

**Rules:** WAL, `synchronous=FULL`, foreign keys, bounded busy timeout, unique idempotency, unique Change claims, transactional HEAD finalization.

### GRF-104 — Local object storage

**Outcome:** SHA-256 objects with shard paths, temporary writes, sync, restrictive modes, atomic rename, and integrity-oriented reads.

**Deferred intentionally:** packs, remote archives, deltas, and repacking.

### GRF-105 — Durable Change acceptance

**Outcome:** versioned API and Studio ingestion. Complete desired state is durably retained before `202 Accepted`.

**Rules:** no target mutation; retries converge on one Change; different requests using one key conflict.

### GRF-106 — Proposal claims and hashing

**Outcome:** explicit ordered selection, current HEAD capture, atomic all-or-nothing claims, deterministic Proposal hash, and Context PR Studio flow.

**Rules:** a Change is claimed by at most one Proposal. Checks and approvals cannot silently migrate between hashes.

### GRF-107 — Evaluation and approvals

**Outcome:** Qdrant preview metadata, deterministic checks, optional structured natural-language evidence, and local actor approval.

**Rules:** current passing evidence is required before approval; model output is evidence, not release authority.

### GRF-108 — Qdrant adapter

**Outcome:** Qdrant Read, Fingerprint, Preview, Compile, Apply, Verify, Restore, and capabilities behind TargetAdapter.

**Rules:** no Qdrant types in Engine. Sparse point release is described as recoverable unless the target can prove atomicity.

### GRF-109 — Release Intent and recovery

**Outcome:** release serialization, HEAD/base validation, before-image capture, persisted Intent, apply, verify, atomic finalization, and startup reconciliation.

**Rules:** only this path mutates Qdrant. HEAD never advances after known apply/verify failure.

### GRF-110 — Forward rollback

**Outcome:** reconstruct older retained values, reduce by logical unit, create rollback Changes, and produce a new Proposal based on current HEAD.

**Rules:** no HEAD rewind. Rollback uses ordinary evaluation, approval, and Release machinery.

### GRF-111 — Studio shell

**Outcome:** Ledgers, Changes, Proposals, and Releases as the only top-level product areas. Evaluation actions are under Proposals; rollback is under Releases.

**Rules:** reusable UI is visual only; domain-aware components stay in feature folders.

### GRF-112 — Single-image runtime

**Outcome:** multi-stage build, embedded Studio assets, one Go HTTP server, `/data` persistence, automatic migration, structured logs, graceful shutdown, and one public port.

**Rules:** Node is build-only. Docker Compose is not required. The image includes llama-server but no model.

### GRF-113 — Optional Gemma evaluation

**Outcome:** InferenceProvider boundary, typed request/result, llama-server loopback HTTP provider, child lifecycle, readiness, structured parsing, and disabled mode.

**Rules:** model path is configurable; no mandatory download; llama-server is not public; deterministic governance works without inference.

## Next production-hardening slices

These are concrete hardening work, not architecture changes.

### GRF-114 — Authentication and secret storage

- Add ingestion API tokens scoped to Ledgers.
- Keep the browser session separate from ingestion credentials.
- Store only token verifiers and server-side Qdrant secret material.
- Add centralized redaction tests.

### GRF-115 — Asynchronous Change preparation

- Preserve current durable acceptance boundary.
- Add persisted preparation attempts and reclaimable worker ownership.
- Read/fingerprint Qdrant after acceptance and before review readiness.
- Keep target I/O outside SQLite transactions.
- Distinguish retryable target outages from permanent invalid Changes.

### GRF-116 — Proposal editing and evidence staleness

- Transactional add/remove membership.
- Recompute hash and mark prior checks/approvals stale.
- Release claims on cancellation.
- Prevent edits once release starts.

### GRF-117 — Release recovery UX

- List unfinished and recovery-required Intents in Release details.
- Provide inspect/retry/resolve operations with explicit target classification.
- Add crash failpoints around every durable release boundary.

### GRF-118 — Retention, quotas, and backup

- Configure pending bytes and rollback-material budgets.
- Keep audit metadata permanent while making rollback payload retention explicit.
- Add consistent SQLite backup plus reachable-object manifest.
- Reject writes before filesystem exhaustion.

### GRF-119 — Complete Qdrant qualification

- Run adapter tests against a real Qdrant instance in CI.
- Exercise payload/vector variants, deletes, API-key auth, target drift, partial failure, and verification mismatch.
- Compare FAST preview with reference queries for supported evaluation policies.

### GRF-120 — Browser end-to-end qualification

- Automate Ledger → Change → Proposal → Evaluation → Approval → Release.
- Automate Release → rollback Proposal → Release.
- Verify persisted volume restart and graceful container shutdown.
- Keep Gemma model tests optional and externally provisioned.

## Quality gate

Every implementation slice must preserve the dependency rules and run:

```text
cd runtime
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/gyrifi

cd ..
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build

docker build -t gyrifi:dev .
```

The release image is then smoke-tested with one port and one persistent volume. A real Qdrant-backed release requires a configured reachable Qdrant collection; local model evaluation additionally requires a user-provided Gemma GGUF.
