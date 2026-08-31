# GRF-232 — Browser end-to-end qualification

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 4 — Qualification |
| Epic | Quality |
| Priority | Medium |
| Size | L |
| Depends on | GRF-205, GRF-206, GRF-207, GRF-208 |
| Blocks | — |

## Summary

Fill the empty `e2e/` directory with browser tests that drive the shipped Docker image through the full governance workflow.

## Context

`e2e/` exists and is empty. Nothing verifies that the built image actually works: that the Go binary serves the embedded Studio bundle, that the SPA fallback route resolves, that data survives a container restart on a mounted volume, or that a real user can complete a release without reading the API docs.

Unit and integration tests cover the parts. This covers the product.

## Scope

### In scope

- Playwright against the built Docker image with a real Qdrant.
- The complete happy path, the rollback path, persistence across restart, and graceful shutdown.
- A small number of high-value journeys — not a mirror of the component suite.

### Out of scope

- Cross-browser matrices. Chromium only.
- Mobile viewports. Studio is a desktop operator tool.
- Gemma-dependent assertions (see notes).
- Performance testing.

## Acceptance criteria

**Harness**

- [x] `e2e/` is its own package with `package.json`, `playwright.config.ts`, and `tests/`. It is **not** added to `pnpm-workspace.yaml` `packages` unless dependency hoisting requires it — decide, and document the decision.
- [x] Scripts use the direct-entry-point form required by the `:` in the workspace path (e.g. `node node_modules/@playwright/test/cli.js test`).
- [x] A `docker compose` file in `e2e/` brings up the built `gyrifi` image plus a pinned `qdrant/qdrant`, with a named volume for `/data`.
- [x] The suite builds the image from the repository `Dockerfile`; it never tests a stale or published image.
- [x] Playwright waits on `/readyz` returning `{"ready":true}` before starting.
- [x] Traces and screenshots are captured on failure and retained as CI artefacts.
- [x] Each test creates its own Ledger and Qdrant collection; tests do not depend on execution order.

**Journeys**

- [x] **Full governance path**, driven entirely through the UI where the UI supports it and through the API only for ingestion (which is the real-world shape): create a Ledger in Studio → `POST` two Changes via the API → see them appear in the Changes inbox → create a Proposal selecting both → run an evaluation with custom criteria → see the evidence rendered → approve → release → confirm the release appears as `HEAD` → verify the points exist in Qdrant with the expected values by querying Qdrant directly.
- [x] **Rollback path**: from the above state, ingest and release a second Proposal, then roll back to the first release, complete the resulting proposal through evaluation and approval and release, and assert Qdrant holds the original values and `HEAD` has moved **forward** to a third release.
- [x] **Gates are enforced in the browser**: `Approve` is disabled before evaluation and shows its reason; `Release` is disabled before approval and shows its reason.
- [x] **Persistence**: restart the container with the same volume, reload Studio, and assert every Ledger, Change, Proposal, and Release is intact and `HEAD` is unchanged.
- [x] **Graceful shutdown**: send `SIGTERM` during idle, assert exit code 0 and no `-wal` growth pathology; restart and assert the database opens cleanly.
- [x] **SPA routing**: deep-link directly to `#changes` and to a proposal detail URL in a fresh page load and assert the correct view renders — this exercises the `studioHandler` fallback.
- [x] **Empty first-run**: a fresh volume shows the Ledgers empty state with its call to action, and nothing errors.

**Discipline**

- [x] Locators use roles and accessible names, never CSS classes or `nth-child`.
- [x] No fixed `waitForTimeout`. Use web-first assertions.
- [x] Each test asserts against Qdrant's actual contents at least once — a UI-only assertion cannot prove the release landed.
- [x] The suite completes in a bounded time and is stable across three consecutive runs.
- [ ] CI (GRF-233) runs the suite; failures block the pipeline. The suite and stable command are ready; workflow enforcement remains deferred because GRF-233 has not landed.

GRF-240 qualification additionally exercises populated Changes and server-disabled Proposal gates at 1440, 1180, 900, and 480 px, asserting that all required controls/reasons remain visible and the document does not overflow horizontally.

## Implementation notes

- **Gemma:** inference is optional and the model file is large. Run the suite with `GYRIFI_INFERENCE_ENABLED=false` by default and assert that the evidence panel correctly reports deterministic-only evaluation. Add a separate, non-required job that enables inference with an externally provisioned model path if one is available. Do not download a model in the default CI path.
- ADR 0002 superseded application authentication, so the suite exercises the trusted-boundary API directly and does not seed credentials.
- Reset strategy: prefer per-test isolation via unique Ledger and collection names over cross-test database truncation.
- Keep the number of tests small. Each one is expensive; each one should fail for a reason that matters.
- Verify Qdrant contents with `fetch` against its REST API from within the test, not with a client library.

## Test plan

The ticket is the test plan. Verification: the suite passes three consecutive clean runs, and deliberately breaking the release finalisation makes the full-path test fail rather than time out.

## Docs to update

- `docs/ai/tech-spec.md` §12/§13 — the e2e invocation and prerequisites.
- `docs/ai/repo-structure.md` — `e2e/` contents and its relationship to the pnpm workspace.
- `README.md` — how to run the e2e suite locally.
- `docs/ai/phases/phase-4.md` — completion entry, including the runtime of the suite and any flakiness found and fixed.

## Definition of done

All acceptance criteria checked, quality gate green including the e2e job, INDEX status updated, phase log entry written.
