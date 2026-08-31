# GRF-226 — Request rate limiting and abuse controls

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Security |
| Priority | Medium |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Bound the request rate a single trusted-network client can impose. ADR 0002 delegates admission and identity to the deployment boundary, so limiting remains availability protection rather than an authentication control.

## Context

`runtime/internal/interfaces/http/server.go` applies request-id, recovery, logging, security headers, and a 4 MiB body cap. There is no rate limiting of any kind.

Why it matters here specifically:

- **Ingestion is the intended high-volume path.** An automated producer with a retry loop and no backoff can saturate the runtime. Because SQLite is configured with `SetMaxOpenConns(1)`, write contention does not degrade gracefully — it serialises, and a flood of writes starves the operator's read requests. Studio becomes unusable exactly when someone needs to look at what is happening.
- **Evaluation is expensive.** Each call runs an LLM inference. A loop calling `POST .../evaluation` will pin CPU and make every other evaluation time out.
- **The deployment boundary is trusted, not infallible.** A compromised or runaway internal workload can still starve operators, so every admitted client is limited by validated client address.

## Scope

### In scope

- A token-bucket limiter keyed by validated client address.
- Per-class limits, with a stricter class for evaluation/release operations.
- Standard rate-limit response headers.
- Studio handling of `429`.

### Out of scope

- Distributed rate limiting. Single replica by design; an in-process limiter is correct.
- A WAF, IP reputation, or geo blocking.
- Per-ledger quotas on stored data — that is GRF-222.
- Any new dependency. `golang.org/x/time/rate` is a reasonable exception to propose in the phase log **only if** hand-rolling proves error-prone; the default expectation is standard library.

## Limit classes

| Class | Routes | Suggested default | Keyed by |
|---|---|---|---|
| `expensive` | `POST .../evaluation`, `POST .../release`, `POST .../rollback` | 10 / minute, burst 3 | client address |
| `ingest` | `POST .../changes` | 100 / second, burst 200 | client address |
| `read` | all `GET` | 200 / second, burst 400 | client address |
| `exempt` | `/healthz`, `/readyz` | unlimited | — |

Defaults are starting points, not commandments. Tune them against a realistic ingestion run and record the final values in the phase log.

## Acceptance criteria

**Mechanism**

- [ ] A limiter middleware in `runtime/internal/interfaces/http/ratelimit.go`, applied before the body cap and handler.
- [ ] Keying uses the validated client address for every limited route.
- [ ] Client address is derived correctly: use `RemoteAddr` by default, and honour `X-Forwarded-For` **only** when `GYRIFI_TRUSTED_PROXIES` is configured and the immediate peer matches. Blindly trusting `X-Forwarded-For` lets any client bypass the limiter by forging a header — this is the classic mistake and a test must cover it.
- [ ] Buckets are evicted after an idle period so memory cannot grow unboundedly from unique keys. Verified by a test that creates many keys and asserts the map shrinks.
- [ ] Eviction and refill are lock-cheap and race-clean under `-race`.
- [ ] `/healthz` and `/readyz` are never limited — an orchestrator's health probes must not be throttled into declaring the process dead.

**Responses**

- [ ] Exceeding a limit returns `429` with the standard error envelope and code `RESOURCE_EXHAUSTED`, plus a `Retry-After` header in seconds.
- [ ] Responses carry `RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset` on every limited route, not only on rejection.
- [ ] The error message states the class and the limit, and never reveals another principal's usage.
- [ ] A `429` is logged at `warn` with the client key **hashed or truncated**, never the raw forwarded header.

**Configuration**

- [ ] Each class is configurable: `GYRIFI_RATELIMIT_EXPENSIVE`, `_INGEST`, `_READ`, in the form `<rate>/<period>:<burst>` (e.g. `100/1s:200`).
- [ ] `GYRIFI_RATELIMIT_ENABLED` (default `true`). Disabling logs a warning at startup.
- [ ] Invalid configuration fails startup with a clear message, consistent with the existing config validation.

**Behaviour**

- [ ] A throttled ingestion client receives `429` and, on retry after `Retry-After`, succeeds — verified end to end.
- [ ] **Rate limiting never causes a partial governance operation.** A `429` is returned before any handler work begins; no Change is written, no intent is created. Assert no side effects after a rejected release.

**Studio**

- [ ] `ApiError` carries the `429` code and `Retry-After`.
- [ ] The UI surfaces "Too many requests — retrying in Ns" and retries **reads** automatically after the indicated delay.
- [ ] Mutations are **never** retried automatically on `429`. The user is shown the delay and re-submits. Auto-retrying an approve or a release is unacceptable.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- A per-key token bucket needs only a timestamp and a float; refill lazily on access rather than with a background ticker. This avoids a goroutine per key.
- Sharding the key map by hash reduces lock contention if a single mutex proves hot. Measure before adding complexity.
- Order matters: request id → recovery → logging → rate limit → body cap → handler. Rate limiting precedes the body cap so a flood of large bodies is rejected cheaply.
- Do not rate-limit `/events/v1` per message; limit *connection establishment* instead. A long-lived SSE connection is one request.
- The `Retry-After` value should be the bucket's actual time-to-token, not a fixed constant.

## Test plan

- `runtime/internal/interfaces/http/ratelimit_test.go`:
  - burst is allowed, the next request is rejected, and it succeeds again after the refill interval (fake clock),
  - each class applies to its own routes with its own limits,
  - keys are isolated: one client address's exhaustion does not affect another,
  - `X-Forwarded-For` is ignored when the peer is not a trusted proxy, and honoured when it is,
  - idle buckets are evicted,
  - `/healthz` and `/readyz` are never limited,
  - concurrent access is race-clean.
- `runtime/tests/ratelimit_side_effects_test.go` — a `429` on `POST .../release` leaves no release intent and no status change.
- Config parsing: valid forms accepted, invalid forms fail startup.
- Studio: `429` on a read auto-retries once after the delay; `429` on a mutation does not.

## Docs to update

- `docs/ai/tech-spec.md` §2 (config keys), §3 (middleware order, `429` / `RESOURCE_EXHAUSTED`, rate-limit headers).
- `docs/ai/product.md` §7 — remove the rate-limiting gap row.
- `docs/ai/tickets/GRF-220-authentication.md` — note that the deferred follow-up now exists.
- `README.md` — tuning guidance for high-volume ingestion.
- `docs/ai/phases/phase-3.md` — completion entry with the tuned limits and the measurements behind them.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
