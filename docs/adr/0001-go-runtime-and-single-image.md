# ADR 0001: Go runtime and single-image distribution

- Status: Accepted
- Date: 2026-08-12

## Context

Gyrifi needs one understandable product boundary while coordinating durable local governance state, a mutable target database, a browser frontend, and optional local model inference. Earlier planning considered a native executable as the canonical distribution and Rust as a possible systems runtime. The settled delivery contract is instead one Docker image and one user-facing application.

The application needs straightforward HTTP/SSE serving, explicit concurrency, reliable process supervision, simple container deployment, and a contribution model accessible to product engineers. Local model execution already has a mature native owner in llama.cpp; duplicating that responsibility in the Gyrifi runtime would create unnecessary CGO and model-runtime coupling.

## Decision

The Gyrifi runtime is implemented in Go as a modular monolith.

- `runtime/cmd/gyrifi` produces one Gyrifi executable.
- `runtime/internal/ledger` is the deterministic, I/O-free governance model.
- `runtime/internal/engine` contains application behavior.
- `Repository`, `TargetAdapter`, and `InferenceProvider` are the three meaningful external boundaries.
- SQLite plus local content-addressed objects is the first Repository implementation.
- Qdrant is the first TargetAdapter.
- llama.cpp is integrated through loopback HTTP to `llama-server`, not through direct `libllama` CGO bindings.
- Gemma GGUF is the initial supported local model family/configuration.
- React, TypeScript, and Vite build Studio. The Go server serves the production assets.
- A multi-stage Dockerfile is the distribution boundary. Node.js, pnpm, and the Go toolchain do not exist as application servers in the final image.

The final image contains the Go executable, Studio assets, migration assets embedded in Go, and llama-server. Local inference remains disabled unless a user mounts a model and enables it.

## Why Go

- Simpler contribution model for a networked local application.
- Strong standard-library HTTP, cancellation, process, logging, and concurrency support.
- Fast, reproducible single-executable builds.
- Good systems performance without making native model bindings part of the runtime.
- Straightforward container lifecycle and graceful shutdown.
- llama.cpp remains responsible for optimized native local inference.

## One container, not artificially one process

Gyrifi optimizes for one product and one container. When local inference is enabled, the Go process supervises a child `llama-server` process inside that container. This is intentional: users still operate one image and expose one Gyrifi port, while the specialized native inference runtime remains isolated behind `InferenceProvider`.

`llama-server` binds only to loopback and is not exposed in image metadata. Go validates its configured GGUF path, starts it, waits for health, and terminates it on shutdown.

## Consequences

### Positive

- One deployment and one public HTTP surface.
- Governance, persistence, target, and inference dependencies remain explicit.
- Deterministic Gyrifi behavior works without a model.
- Target and model details do not leak into the Ledger or Engine.
- No Rust, Python runtime, production Node server, microservice coordinator, or CGO llama binding is required.

### Trade-offs

- The image contains both Go and llama.cpp runtime artifacts, even when inference is disabled.
- A mounted GGUF can be large; models are deliberately not baked into the base image.
- Qdrant's sparse update behavior is recoverable but not universally atomic, so Gyrifi must retain Release Intent reconciliation.
- Multiple Gyrifi replicas against one data directory are outside this architecture.

## Rejected alternatives

- **Rust runtime:** strong systems language, but not the settled contribution and orchestration choice.
- **Python inference service:** adds another runtime and service boundary for behavior llama.cpp already owns.
- **Production Node server:** duplicates the Go HTTP surface; Vite output is static.
- **Direct libllama binding:** introduces CGO and couples Go lifecycle to model internals.
- **Docker Compose/microservices:** unnecessary for one product and one target adapter.
- **Native installers as the primary distribution:** may be revisited, but Docker is the current explicit distribution boundary.
