# GRF-224 — Health, readiness, and operational metrics

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Operations |
| Priority | High |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Add liveness and readiness endpoints and a minimal metrics surface, so an orchestrator can tell whether the runtime is alive, whether it should receive traffic, and what it is doing.

## Context

The only health-adjacent endpoint is `GET /api/v1/system/status`, which returns version and adapter names. Verified problems:

- **It is not a readiness signal.** It answers 200 while the runtime is in states where it should not receive traffic.
- **Recovery failure is invisible to an orchestrator.** In `runtime/internal/bootstrap/bootstrap.go`:
  ```go
  if err := application.RecoverReleases(ctx); err != nil {
      logger.Error("release recovery needs attention", "error", err)
  }
  ```
  Startup continues. That is the right call — the operator needs the API up to *diagnose* the problem via GRF-213 — but nothing distinguishes "healthy" from "running with unresolved release intents" to anything outside the process.
- **GRF-232 and GRF-233 both poll `/api/v1/system/status`** as a stand-in for a health check, because there is nothing better.
- **No metrics of any kind.** Structured `slog` JSON logging exists and is good (`bootstrap.go` configures a JSON handler with a `GYRIFI_LOG_LEVEL`-driven level, and the HTTP middleware logs method, path, status, duration, and request id). But there is no way to answer "how many releases failed this week" without grepping logs.

## Scope

### In scope

- `GET /healthz` — liveness.
- `GET /readyz` — readiness, with a machine-readable reason when not ready.
- `GET /metrics` — Prometheus text format, from the standard library.
- Counters and histograms for the operations that matter.

### Out of scope

- A metrics client library. Prometheus text format is trivial to emit and this runtime has a one-dependency policy.
- Distributed tracing.
- Alerting rules or dashboards.
- Changing the existing logging, which is already structured and adequate.

## Semantics — the part that matters

| Endpoint | Answers | Fails when |
|---|---|---|
| `/healthz` | "Is this process alive and not deadlocked?" | Never, unless the process cannot serve at all |
| `/readyz` | "Should this process receive traffic?" | The database is unreachable, migrations have not completed, or the process is shutting down |

**A `RECOVERY_REQUIRED` release intent must NOT make `/readyz` fail.** Draining traffic from the only replica that can be used to *resolve* the problem would turn a recoverable incident into an outage. It is reported as a distinct signal instead.

This distinction is the single most important thing in this ticket. Get it wrong and an operator loses their ability to fix a failed release.

## Acceptance criteria

**Liveness**

- [x] `GET /healthz` returns `200` with body `ok` as soon as the HTTP server is listening.
- [x] It performs **no** database query and takes no locks. A slow or locked database must not make the process look dead and get it killed.
- [x] It is unauthenticated even after GRF-220, and is exempt from rate limiting (GRF-226).

**Readiness**

- [x] `GET /readyz` returns `200` with `{"ready":true}` when the runtime can serve, and `503` with `{"ready":false,"reasons":["..."]}` when it cannot.
- [x] It verifies database reachability with a bounded, trivial query (e.g. `SELECT 1`) with a short timeout, and reports `database_unreachable` on failure.
- [x] It reports `migrations_incomplete` if schema migration has not finished.
- [x] It returns `503` with `shutting_down` from the moment graceful shutdown begins, so a load balancer drains before connections are closed.
- [x] It **does not** consider release intents, target reachability, or inference state. Those are dependencies of specific operations, not of the process's ability to serve.
- [x] Result is not cached beyond 1 second.

**Status endpoint enrichment**

- [x] `GET /api/v1/system/status` gains `"health": { "database": "ok", "target": "ok|unreachable|unknown", "inference": "ok|disabled|unhealthy", "unresolvedIntents": <int> }`.
- [x] Target and inference checks are **cached** (default 15s) and never block the request. A slow Qdrant must not make the status endpoint slow.
- [x] `unresolvedIntents` counts `RECOVERY_REQUIRED` intents across all ledgers and is the signal Studio's recovery banner (GRF-208) and any external alerting should use.

**Metrics**

- [x] `GET /metrics` emits Prometheus text format v0.0.4 with correct `# HELP` and `# TYPE` lines.
- [x] Implemented in `runtime/internal/interfaces/http/metrics.go` using only the standard library. No new dependency.
- [x] Counters: `gyrifi_http_requests_total{method,path_template,status}`, `gyrifi_changes_accepted_total{ledger}`, `gyrifi_proposals_created_total`, `gyrifi_evaluations_total{passed}`, `gyrifi_releases_total{outcome}`, `gyrifi_rollbacks_total`, `gyrifi_target_requests_total{operation,outcome}`.
- [x] Gauges: `gyrifi_unresolved_intents`, `gyrifi_object_store_bytes`, `gyrifi_pending_changes`, `gyrifi_build_info{version,commit}` (value 1).
- [x] Histogram: `gyrifi_http_request_duration_seconds{path_template}` with sane buckets.
- [x] **`path_template` is the route pattern, never the raw path.** Emitting `/api/v1/ledgers/ldg_abc123/changes` as a label produces unbounded cardinality and will take down a Prometheus server. A test asserts that ids never appear in label values.
- [x] No label carries a ledger name, unit id, hash, actor, token, or any other unbounded or sensitive value. Ledger **ids** on the changes counter are acceptable only if bounded; if a deployment can have many ledgers, drop the label — decide and document.
- [x] Metric collection is lock-cheap: `sync/atomic` counters, not a mutex per request.
- [x] `/metrics` requires authentication once GRF-220 lands, or is bindable to a separate loopback-only address via `GYRIFI_METRICS_ADDRESS`. Decide, and document the reasoning.

**General**

- [x] `/healthz`, `/readyz`, and `/metrics` are registered outside `/api/v1` and are excluded from the SPA fallback in `studioHandler` — which currently only excludes `/api/` and `/events/` prefixes. Verify a request to `/healthz` is not served `index.html`.
- [x] `go test ./...` passes; `-race` clean.

## Implementation notes

- Register the three routes in `routes()` and extend the `studioHandler` prefix guard. This is easy to miss and produces a confusing bug: `/healthz` returning the Studio HTML with a `200`, which looks healthy to a naive check.
- Readiness during shutdown requires a flag set before `server.Shutdown` is called in `bootstrap.go`, plus a short sleep before shutdown so the load balancer observes the `503`. Make that drain delay configurable (`GYRIFI_DRAIN_DELAY`, default `0` for local, documented for orchestrated deployments).
- The Prometheus text format is line-oriented and simple; write it directly. Escaping rules for label values are the only fiddly part — quote and escape `\`, `"`, and newline.
- Increment counters in the engine, not the handlers, for domain events. HTTP counters belong in middleware.
- Reuse the cached target/inference health for both the status endpoint and the metrics gauges — do not probe twice.

## Test plan

- `runtime/internal/interfaces/http/health_test.go`:
  - `/healthz` returns 200 with no database available (inject a repository that errors),
  - `/readyz` returns 503 with `database_unreachable` when the database fails,
  - `/readyz` returns 503 with `shutting_down` once the shutdown flag is set,
  - `/readyz` returns 200 with a `RECOVERY_REQUIRED` intent present — the explicit anti-regression test for the semantics above,
  - `/healthz` is not served the SPA fallback.
- `runtime/internal/interfaces/http/metrics_test.go`:
  - output parses as valid Prometheus text format,
  - `path_template` labels contain no id-shaped segments across a request to every registered route,
  - counters increment correctly through a full governance flow,
  - concurrent requests are race-clean under `-race`.
- Status endpoint returns cached target health without blocking when the target is unreachable (assert a bounded response time).

## Docs to update

- `docs/ai/tech-spec.md` §2 (new config keys), §3 (three new endpoints, enriched status payload, and the readiness semantics table).
- `docs/ai/product.md` §7 — remove the observability gap row.
- `docs/ai/tickets/GRF-232-e2e-suite.md` and `GRF-233-ci-pipeline.md` — switch their readiness polling from `/api/v1/system/status` to `/readyz`.
- `README.md` — health endpoints and a note on scraping metrics.
- `docs/ai/phases/phase-3.md` — completion entry, including the final decision on metrics endpoint exposure.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
