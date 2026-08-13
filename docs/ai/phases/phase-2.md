# Phase 2 — Governance API completeness

**Goal:** close the gaps where the runtime holds state it will not let anyone read, or reaches a state it will not let anyone leave. Every one of these tickets fixes a place where the audit trail exists in the database but not in the API.

**Status:** Not started

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-210](../tickets/GRF-210-event-stream.md) | Real domain event stream | M | — | Not started |
| [GRF-211](../tickets/GRF-211-proposal-detail-api.md) | Proposal detail and evidence read API | M | — | Not started |
| [GRF-212](../tickets/GRF-212-proposal-cancellation.md) | Proposal cancellation and claim release | M | — | Not started |
| [GRF-213](../tickets/GRF-213-release-intent-api.md) | Release intent inspection and recovery API | L | — | Not started |
| [GRF-214](../tickets/GRF-214-pagination.md) | Pagination and filtering for list endpoints | M | — | Not started |
| [GRF-215](../tickets/GRF-215-lifecycle-management.md) | Ledger and Change lifecycle management | M | GRF-212 | Not started |

## Phase-level notes

- **GRF-211 and GRF-213 are prerequisites for Phase 1 work** (GRF-207 and GRF-208 respectively). Schedule them accordingly — the recommended order in [INDEX.md](../tickets/INDEX.md) interleaves the phases for this reason.
- This phase introduces the first migrations after `001_initial.sql`. `001_initial.sql` is frozen; every schema change is a new numbered file. Migration numbers are allocated in ticket order: 002 (GRF-212), 003 (GRF-213), 004 (GRF-214), 007 (GRF-215). If tickets land out of order, renumber to match the actual apply order and record it here.
- Three tickets add enum members — `ProposalStatus.CANCELLED` (GRF-212), `ReleaseIntentStatus.ABANDONED` (GRF-213), and `ChangeStatus.WITHDRAWN` (GRF-215). All must be reflected in `design-system.md` §2.2 status tone mapping and in the exhaustive TypeScript mapping, or the frontend build breaks. That breakage is intentional and desirable.
- GRF-212 and GRF-215 both manipulate `proposal_changes` claims. Do GRF-212 first; GRF-215's withdrawal guard depends on cancellation existing as the escape hatch for a claimed Change.
- No new Go dependencies are permitted in this phase.

## Exit criteria

- [ ] All six tickets complete.
- [ ] No state the runtime records is unreadable through the API.
- [ ] No state the runtime can enter is unresolvable through the API.
- [ ] No mistake a user can make is irreversible except by design.
- [ ] Every list endpoint is bounded.
- [ ] `/events/v1` carries real domain events and Studio no longer polls.
- [ ] `go test ./...` green, including new migration tests.

## Completed entries

_No entries yet. Use the template in [README.md](README.md) and append newest last._
