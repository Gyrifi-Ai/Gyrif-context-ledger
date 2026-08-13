# Phase 3 — Production hardening

**Goal:** make Gyrifi safe to run somewhere other than a developer's laptop. Everything here is a property the product currently lacks that would make a real deployment irresponsible.

**Status:** Not started

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-223](../tickets/GRF-223-build-metadata.md) | Build metadata and version consistency | S | — | Not started |
| [GRF-221](../tickets/GRF-221-change-preparation.md) | Asynchronous Change preparation and base fingerprint | L | — | Not started |
| [GRF-222](../tickets/GRF-222-retention-backup.md) | Retention budgets, quotas, and backup command | L | — | Not started |
| [GRF-224](../tickets/GRF-224-health-and-metrics.md) | Health, readiness, and operational metrics | M | — | Not started |
| [GRF-225](../tickets/GRF-225-inference-supervision.md) | Inference process supervision | M | — | Not started |
| [GRF-220](../tickets/GRF-220-authentication.md) | Ingestion tokens and browser session auth | XL | — | Not started |
| [GRF-226](../tickets/GRF-226-rate-limiting.md) | Request rate limiting and abuse controls | M | GRF-220 | Not started |

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

_No entries yet. Use the template in [README.md](README.md) and append newest last._
