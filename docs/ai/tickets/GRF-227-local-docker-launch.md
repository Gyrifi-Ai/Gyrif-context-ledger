# GRF-227 — Local Docker launch

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 3 — Production hardening |
| Epic | Operations |
| Priority | High |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Make the supported local product launch a single VS Code Run and Debug action or `docker compose up --build` command. The launch must build Gyrifi, start its required local Qdrant target, provision the development collection without replacing existing data, and expose Studio only on loopback.

## Context

The shipping Docker image already packages the Go runtime and compiled Studio, but a person currently has to build it manually and provide a separately configured Qdrant endpoint. That is an unnecessary barrier for local evaluation and makes the first run look broken when the default collection does not exist.

This ticket is intentionally local-only. GRF-220 has not landed, so no unauthenticated service may be exposed beyond `127.0.0.1`.

## Scope

### In scope

- A pinned Docker Compose topology for the Gyrifi image and a local Qdrant container.
- Durable named volumes for Gyrifi state and Qdrant data.
- Idempotent creation of the `gyrifi` development collection with a three-dimensional cosine vector configuration.
- A repository-owned VS Code Run and Debug configuration that launches Compose with a single action.
- README and reference documentation for the supported local launch and lifecycle.

### Out of scope

- Production deployment, TLS, authentication, remote Qdrant, or container orchestration.
- Changing the Qdrant adapter, its collection model, or governance workflow.
- Bundling Qdrant into the Gyrifi application image.

## Acceptance criteria

- [x] `docker compose up --build` builds and starts the complete local stack without a separate runtime or Studio process.
- [x] The VS Code Run and Debug control starts the same Compose command from the repository root.
- [x] Studio is reachable at `http://127.0.0.1:8080`; no service port is published to non-loopback interfaces.
- [x] Gyrifi state and Qdrant state survive `docker compose down` and restart through named volumes.
- [x] The collection initializer waits for Qdrant readiness, preserves an existing `gyrifi` collection, and creates it as a 3-dimensional cosine collection only when absent.
- [x] The Compose file pins all external image versions and does not introduce an application dependency.
- [x] Documentation states that this is local-only until GRF-220 authenticates the runtime, and gives the shutdown/reset commands.

## Test plan

- Validate the Compose model with `docker compose config`.
- Start the stack from a clean state, request the system status endpoint, and confirm the collection exists through the Qdrant API.
- Run the repository quality gate.

## Docs to update

- `README.md` — primary local launch and shutdown/reset instructions.
- `docs/ai/tech-spec.md` §1–§2 — local composition contract.
- `docs/ai/repo-structure.md` §1 and §4 — Compose and VS Code launch files.
- `docs/ai/phases/phase-3.md` — completion entry with actual verification output.

## Definition of done

All acceptance criteria met, full quality gate green, INDEX status updated, and a Phase 3 log entry written.
