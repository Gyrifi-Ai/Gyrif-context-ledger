# Implementation backlog

Every ticket is a self-contained work order. **Read the ticket file, not this index, before implementing.** This index exists for selection and sequencing only.

Ticket files: `docs/ai/tickets/GRF-2XX-*.md`. Working agreement: [AGENTS.md](../../../AGENTS.md).

## Numbering

IDs are allocated in blocks by phase, **not** contiguously. Gaps in the sequence are intentional.

| Block | Phase | Allocated | Free |
|---|---|---|---|
| GRF-201 … GRF-209 | Phase 1 — Studio experience | 201–209 | **none — block full** |
| GRF-210 … GRF-219 | Phase 2 — Governance API completeness | 210–215 | 216, 217, 218, 219 |
| GRF-220 … GRF-229 | Phase 3 — Production hardening | 220–226 | 227, 228, 229 |
| GRF-230 … GRF-239 | Phase 4 — Qualification | 230–233 | 234 … 239 |

A missing number means "not yet allocated", never "lost ticket". Every allocated ID has a file in this directory and a row in the status log below; the two lists must always match, and a script-friendly check of that is in the AGENTS.md quality gate.

**The phase of a ticket is whatever the tables below and the ticket's own metadata say — not what its number implies.** The blocks are a filing convenience, and this index is the source of truth.

**Overflow rule.** The Phase 1 block is exhausted. If new Phase 1 work appears, allocate from **GRF-240 upward** and list it in the Phase 1 table with its real phase. Do not renumber existing tickets to make room; stable IDs matter more than a tidy sequence.

### Parked work — candidates for the free slots

These are **not** tickets. They are scope that existing tickets explicitly pushed out, recorded here so the free numbers are not a mystery and so nobody re-litigates a decision that was already made. Promote one to a ticket only when it is actually needed.

| Parked item | Deferred by | Would sit in |
|---|---|---|
| Editing Proposal membership in place (instead of cancel-and-recreate) | GRF-212 | Phase 2 |
| `REVIEWED` / `APPROVED` Proposal status transitions — currently defined but never written | GRF-211 | Phase 2 |
| Event persistence, replay, and `Last-Event-ID` resume | GRF-210 | Phase 2 |
| Full-text search and configurable sort on list endpoints | GRF-214 | Phase 2 |
| Hard deletion / erasure for a regulatory request | GRF-215 | Phase 2 |
| OIDC / SSO / multi-tenant identity | GRF-220 | Phase 3 |
| Per-Ledger target configuration (collection is process-global today) | product.md §2 | Phase 3 |
| Additional target adapters beyond Qdrant | — | Phase 3 |
| Remote backup destinations and point-in-time recovery | GRF-222 | Phase 3 |
| Mark-and-sweep retention, if the reachable set outgrows memory | GRF-222 | Phase 3 |
| `golangci-lint` adoption and configuration | GRF-233 | Phase 4 |
| Cross-browser and mobile e2e coverage | GRF-232 | Phase 4 |

---

### Phase 1 — Studio experience

Goal: turn the functional-but-plain Studio into a product that reads like a control room. Spec: [design-system.md](../design-system.md).

| ID | Title | Size | Depends on |
|---|---|---|---|
| [GRF-240](GRF-240-mockup-led-studio-product-system.md) | Mockup-led Studio product system | XL | — |
| [GRF-201](GRF-201-design-tokens.md) | Mockup-led design token foundation | M | — |
| [GRF-202](GRF-202-ui-library.md) | UI primitive and pattern library | L | GRF-201 |
| [GRF-203](GRF-203-application-shell.md) | Application shell, navigation, and real runtime status | M | GRF-201, GRF-202 |
| [GRF-204](GRF-204-async-data-layer.md) | Async data layer with loading, error, and empty states | M | GRF-202 |
| [GRF-205](GRF-205-ledgers-page.md) | Ledgers page redesign | M | GRF-203, GRF-204 |
| [GRF-206](GRF-206-changes-page.md) | Changes page redesign | L | GRF-203, GRF-204 |
| [GRF-207](GRF-207-proposals-workspace.md) | Proposals review workspace | XL | GRF-203, GRF-204, GRF-211 |
| [GRF-208](GRF-208-releases-timeline.md) | Releases timeline and rollback flow | L | GRF-203, GRF-204, GRF-213 |
| [GRF-209](GRF-209-studio-resilience.md) | Studio resilience: error boundary, offline state, stream reconnection | M | GRF-202, GRF-204 |

## Phase 2 — Governance API completeness

Goal: close the API gaps that Phase 1 screens depend on, without changing any invariant.

| ID | Title | Size | Depends on |
|---|---|---|---|
| [GRF-210](GRF-210-event-stream.md) | Real domain event stream | M | — |
| [GRF-211](GRF-211-proposal-detail-api.md) | Proposal detail and evidence read API | M | — |
| [GRF-212](GRF-212-proposal-cancellation.md) | Proposal cancellation and claim release | M | — |
| [GRF-213](GRF-213-release-intent-api.md) | Release intent inspection and recovery API | L | — |
| [GRF-214](GRF-214-pagination.md) | Pagination and filtering for list endpoints | M | — |
| [GRF-215](GRF-215-lifecycle-management.md) | Ledger and Change lifecycle management | M | GRF-212 |

## Phase 3 — Production hardening

Goal: make a shared deployment defensible.

| ID | Title | Size | Depends on |
|---|---|---|---|
| [GRF-220](GRF-220-authentication.md) | Trusted deployment boundary decision | XL | — |
| [GRF-221](GRF-221-change-preparation.md) | Asynchronous Change preparation and base fingerprint | L | — |
| [GRF-222](GRF-222-retention-backup.md) | Retention budgets, quotas, and backup command | L | — |
| [GRF-223](GRF-223-build-metadata.md) | Build metadata and version consistency | S | — |
| [GRF-224](GRF-224-health-and-metrics.md) | Health, readiness, and operational metrics | M | — |
| [GRF-225](GRF-225-inference-supervision.md) | Inference process supervision | M | — |
| [GRF-226](GRF-226-rate-limiting.md) | Request rate limiting and abuse controls | M | — |
| [GRF-227](GRF-227-local-docker-launch.md) | Local Docker launch | M | — |

## Phase 4 — Qualification

Goal: prove it works, continuously.

| ID | Title | Size | Depends on |
|---|---|---|---|
| [GRF-230](GRF-230-studio-tests.md) | Studio component and integration test suite | L | GRF-202 |
| [GRF-231](GRF-231-qdrant-qualification.md) | Qdrant integration qualification | L | — |
| [GRF-232](GRF-232-e2e-suite.md) | Browser end-to-end qualification | L | GRF-205…208 |
| [GRF-233](GRF-233-ci-pipeline.md) | CI pipeline | M | — |

---

## Recommended order

```text
GRF-233 → GRF-223 → GRF-201 → GRF-202 → GRF-204 → GRF-203 → GRF-209 → GRF-205
       → GRF-210 → GRF-211 → GRF-206 → GRF-207 → GRF-213 → GRF-208
       → GRF-212 → GRF-215 → GRF-214 → GRF-224 → GRF-225
       → GRF-230 → GRF-231 → GRF-232
       → GRF-221 → GRF-222 → GRF-226
```

Rationale: CI and version metadata first so everything after is verified. Tokens and the component library unblock every screen, and GRF-209 lands immediately after them so every subsequent screen is built on a shell that fails visibly. The two API tickets that Phase 1 screens hard-depend on (GRF-211, GRF-213) land just before the screens that need them. GRF-215 follows GRF-212 because withdrawal and cancellation share the claim-release mechanics. ADR 0002 closed GRF-220 without application auth; GRF-226 now keys limits by validated client address.

## Sizes

| Size | Meaning |
|---|---|
| S | One file, no new interfaces |
| M | A few files in one layer |
| L | Crosses layers, needs new tests |
| XL | Crosses layers, changes a contract, needs a migration or an ADR |

## Status log

Update this table when a ticket is completed, and write the corresponding entry in `docs/ai/phases/phase-N.md`.

| ID | Status | Completed | Phase log entry |
|---|---|---|---|
| GRF-240 | Done | 2026-08-31 | phase-1.md |
| GRF-201 | Done | 2026-08-17 | phase-1.md |
| GRF-202 | Done | 2026-08-17 | phase-1.md |
| GRF-203 | Done | 2026-08-17 | phase-1.md |
| GRF-204 | Done | 2026-08-31 | phase-1.md |
| GRF-205 | Done | 2026-08-31 | phase-1.md |
| GRF-206 | Done | 2026-08-31 | phase-1.md |
| GRF-207 | Done | 2026-08-31 | phase-1.md |
| GRF-208 | Done | 2026-08-31 | phase-1.md |
| GRF-209 | Done | 2026-08-31 | phase-1.md |
| GRF-210 | Done | 2026-08-31 | phase-2.md |
| GRF-211 | Done | 2026-08-31 | phase-2.md |
| GRF-212 | Done | 2026-08-31 | phase-2.md |
| GRF-213 | Done | 2026-08-31 | phase-2.md |
| GRF-214 | Done | 2026-09-01 | phase-2.md |
| GRF-215 | Not started | — | — |
| GRF-220 | Done | 2026-09-01 | phase-3.md |
| GRF-221 | Not started | — | — |
| GRF-222 | Not started | — | — |
| GRF-223 | Done | 2026-08-31 | phase-3.md |
| GRF-224 | Done | 2026-08-31 | phase-3.md |
| GRF-225 | Not started | — | — |
| GRF-226 | Not started | — | — |
| GRF-227 | Done | 2026-08-17 | phase-3.md |
| GRF-230 | Done | 2026-08-31 | phase-4.md |
| GRF-231 | Not started | — | — |
| GRF-232 | Done | 2026-08-31 | phase-4.md |
| GRF-233 | Done | 2026-08-31 | phase-4.md |
