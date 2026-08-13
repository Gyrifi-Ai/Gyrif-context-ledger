# Phase 4 — Qualification

**Goal:** replace assumption with evidence. The runtime's correctness is currently asserted by unit tests against fakes and by nothing else. This phase builds the machinery that proves the shipped artefact does what the documentation claims.

**Status:** Not started

## Tickets

| ID | Title | Size | Depends on | Status |
|---|---|---|---|---|
| [GRF-233](../tickets/GRF-233-ci-pipeline.md) | CI pipeline | M | — | Not started |
| [GRF-230](../tickets/GRF-230-studio-tests.md) | Studio component and integration test suite | L | GRF-202 | Not started |
| [GRF-231](../tickets/GRF-231-qdrant-qualification.md) | Qdrant integration qualification | L | — | Not started |
| [GRF-232](../tickets/GRF-232-e2e-suite.md) | Browser end-to-end qualification | L | GRF-205 … GRF-208 | Not started |

## Phase-level notes

- **GRF-233 is first, ahead of every ticket in every phase.** Each ticket's definition of done requires a green quality gate; without CI that is an honour system. It is the first ticket in the recommended order in [INDEX.md](../tickets/INDEX.md) for this reason.
- GRF-233 ships `integration` and `e2e` job stubs, disabled. GRF-231 and GRF-232 enable them and make them required checks. Enabling is part of those tickets, not a follow-up.
- GRF-231 may discover that the adapter's assumptions about Qdrant are wrong. That is the point. Any such discovery is a finding to record in `tech-spec.md` §9 and, if it is a defect, to fix inside the ticket.
- Inference (Gemma) is deliberately excluded from required CI paths. Model files are large and the product works without them. Any inference-dependent job is advisory and externally provisioned.
- New dependencies are permitted in this phase, confined to test tooling: Testing Library and jsdom (GRF-230), Playwright (GRF-232), coverage provider (GRF-230). Nothing enters the runtime dependency graph.

## What each ticket proves

| Ticket | Claim it makes verifiable |
|---|---|
| GRF-233 | Every commit satisfies the documented quality gate |
| GRF-230 | The UI enforces the governance gates it displays |
| GRF-231 | The adapter reads back exactly what it wrote to a real Qdrant |
| GRF-232 | The shipped image completes the full workflow and survives restart |

The four together mean the audit trail can be trusted end to end. Any one missing leaves a link unverified.

## Exit criteria

- [ ] All four tickets complete.
- [ ] CI runs the full gate plus integration and e2e as required checks.
- [ ] Frontend coverage thresholds met and enforced.
- [ ] The Qdrant adapter is verified against a pinned real Qdrant, including the cosine normalisation tolerance and partial-failure behaviour.
- [ ] The built Docker image is verified to complete a full release and rollback and to persist across a restart.
- [ ] Three consecutive clean runs of the e2e suite pass without flakiness.

## Completed entries

_No entries yet. Use the template in [README.md](README.md) and append newest last._
