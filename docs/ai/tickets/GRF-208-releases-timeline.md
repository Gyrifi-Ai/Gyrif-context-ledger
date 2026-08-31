# GRF-208 — Releases timeline and rollback flow

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | L |
| Depends on | GRF-203, GRF-204, GRF-213 |
| Blocks | GRF-232 |

## Summary

Rebuild `studio/src/features/releases/releases-page.tsx` to the timeline design in [design-system.md §5.4](../design-system.md), including a recovery banner and a rollback flow that explains what rollback actually does.

## Context

Current page renders a `.timeline` of `<article>` elements with `HEAD`/`HISTORY` labels, the release id as an `h3`, the hash in a `<code>`, the proposal id as plain text, and a `Create rollback Proposal` button on non-HEAD entries.

Problems:

- **The rollback button is mislabelled in intent.** Users read "rollback" as "revert now". It actually creates a proposal that must then be evaluated, approved, and released. Nothing on screen says so.
- There is no confirmation and no indication of how many units would be restored.
- The release plan — the operations, expected fingerprints, and whether before-images were retained — is completely invisible, so users cannot tell whether rollback material still exists.
- `RECOVERY_REQUIRED` release intents are invisible. A failed release leaves the user with an error toast and no path forward.
- The page navigates to `#proposals` on success with no explanation of what was created.

## Scope

### In scope

- Timeline of releases with a distinct `HEAD` node.
- Recovery banner driven by the intents API (GRF-213).
- `View plan` drawer showing the compiled operations.
- Rollback behind a `ConfirmDialog` that explains the forward-history semantics.
- Post-rollback handoff to the created proposal.

### Out of scope

- Resolving a recovery-required intent from the UI beyond calling the GRF-213 endpoints — do not invent recovery semantics here.
- Diffing two releases.

## Acceptance criteria

**Timeline**

- [x] `PageHeader` with eyebrow `IMMUTABLE HISTORY`, title `Releases`, and the design-system §5.4 description.
- [x] Releases render newest-first in the `Timeline` pattern. The first entry is marked `HEAD` with the design-system's current orange filled node and a `HEAD` badge; the rest are hollow nodes.
- [x] Each entry shows: `HashChip` for the release id, the source proposal title, `{n} units`, `HashChip` for the release hash, and relative age with an absolute `title`.
- [x] Each entry has `View plan`, and non-HEAD entries additionally have `Roll back to here` (`danger` variant).

**Plan drawer**

- [x] `View plan` opens a `Drawer` listing the operations from the release's intent: unit, action, expected fingerprint, desired fingerprint, and a "before-image retained" indicator.
- [x] When a required before-image is missing for an operation, that row is flagged amber with the text "No rollback material for this unit." An absent pre-state is correctly described as a rollback delete instead.
- [x] The drawer states the target metric when present (e.g. `Cosine`).

**Rollback**

- [x] `Roll back to here` opens a `ConfirmDialog` whose body states, explicitly:
  1. this creates a **new proposal**, it does not rewind history;
  2. `{n}` units will be restored to their state at this release;
  3. the proposal must be evaluated, approved, and released like any other;
  4. `HEAD` will move **forward** to a new release.
- [x] The dialog shows the unique unit count computed from the plans of all releases newer than the target.
- [x] Confirming calls `api.rollback`, and on success shows a success panel with the new proposal's title and a `Review proposal` button that navigates to `#proposals/{id}`.
- [x] A `409` ("The selected Release is already HEAD.") or `500` ("Rollback material is unavailable." / "Retained rollback value is unavailable.") renders the server message verbatim in an `ErrorState` inside the dialog, and the dialog stays open.

**Recovery**

- [x] When the intents endpoint reports ≥1 `RECOVERY_REQUIRED` intent for the ledger, an amber banner renders above the timeline: "{n} release intent(s) require recovery." with an `Inspect` action.
- [x] `Inspect` opens a drawer listing each affected intent: id, proposal, status, created time, and the plan, plus the `Retry verification` / `Mark resolved` actions exposed by GRF-213.
- [x] The banner is absent when there are none. It loads once with the workspace and refetches on advisory domain events rather than polling.

**General**

- [x] All five interaction states. Empty state explains that releases appear here after a proposal is approved and released, with a link to `#proposals`.
- [x] Full keyboard path including the native dialog focus trap, Escape close, and no destructive autofocus.
- [x] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- **This ticket needs GRF-213** for the recovery banner and the plan data. The `Plan` is stored on the `ReleaseIntent`, which has no read endpoint today. Do not scrape it from anywhere else.
- Unit counts for the rollback dialog: prefer a server-provided preview from GRF-213 if available; otherwise compute from the plans returned by the intents endpoint. Never guess.
- Split the page: `releases-page.tsx`, `release-timeline.tsx`, `plan-drawer.tsx`, `rollback-dialog.tsx`, `recovery-banner.tsx`.
- Copy matters here more than anywhere else in the product. Have the rollback dialog text reviewed against [product.md §4](../product.md) before shipping — it must not imply a rewind.
- Reuse `formatAge` from `features/shared/time.ts` (GRF-206).

## Test plan

- `features/releases/releases-page.test.tsx` — HEAD marking, ordering, empty/error/loading states.
- `features/releases/rollback-dialog.test.tsx` — dialog text contains the four required statements; `409` keeps the dialog open with the server message; success navigates to the new proposal.
- `features/releases/recovery-banner.test.tsx` — renders only when intents are present; count is correct.
- `features/releases/plan-drawer.test.tsx` — missing before-image rows are flagged.
- Manual: force an apply failure (point Qdrant at an unreachable URL mid-release) and confirm the recovery banner appears after reload.

## Docs to update

- `docs/ai/design-system.md` §5.4 — mark implemented.
- `docs/ai/product.md` §4 and §7 — record that rollback is now explained in-product; remove the recovery-invisible gap row once GRF-213 and this ticket are both done.
- `docs/ai/phases/phase-1.md` — completion entry, including the final rollback dialog copy verbatim.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
