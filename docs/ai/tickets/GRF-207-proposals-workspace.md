# GRF-207 — Proposals review workspace

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | Highest |
| Size | XL |
| Depends on | GRF-203, GRF-204, GRF-211 |
| Blocks | GRF-232 |

## Summary

Replace the Proposals list-with-three-buttons with the two-pane review workspace in [design-system.md §5.3](../design-system.md). This is the screen where the product's value is either obvious or invisible.

## Context

`studio/src/features/proposals/proposals-page.tsx` today:

- Left panel: a list of `<article>` cards, each showing a status badge, the title, `hash + change count` in a `<code>`, and three small buttons — `Evaluate`, `Approve`, `Release`.
- Right panel: a create form with a `title` input and a scrollable checkbox list (`max-height: 180px`) of `READY` changes.
- **Evaluation criteria is a hardcoded string in the component.** The user cannot express what they want checked.
- Evaluation results are discarded — `api.evaluate` returns `{ passed, summary }` and nothing is rendered.
- `findings` and `previewFidelity` are returned by the server and never shown.
- All three buttons are always enabled; a rejected action just throws.
- Nothing communicates the governance sequence, the bound proposal hash, or why an action is unavailable.

## Scope

### In scope

- Two-pane layout: proposal list (380 px) + detail pane.
- Four-step progress rail: Changes → Evidence → Approval → Release.
- Collapsible detail sections for Changes, Evidence, Approval, Release.
- User-authored evaluation criteria with presets and per-proposal persistence.
- Full evidence rendering including findings, model identity, preview fidelity, and the bound hash.
- Explicit gate reasons on every disabled action.
- Stale-evidence banner.
- Release behind a `ConfirmDialog`.
- Proposal creation drawer with ordered selection.

### Out of scope

- Editing proposal membership after creation (no API; would change the hash — see GRF-212 for cancellation).
- Cancelling a proposal (GRF-212) — leave a disabled `Cancel` with the note "Available in GRF-212".

## Acceptance criteria

**Layout and list**

- [ ] `PageHeader` with eyebrow `CONTEXT PRs`, title `Proposals`, the design-system §5.3 description, and a `+ New proposal` primary action.
- [ ] Left list shows each proposal's status badge (tone from `features/shared/status.ts`), title, `{n} changes`, and relative age. The selected item has a jade left rail.
- [ ] Selecting a proposal updates the detail pane and the URL hash (e.g. `#proposals/pr_7c1e…`) so the view is linkable and survives reload.

**Detail header**

- [ ] Shows title, status badge, `HashChip` for the proposal id, `HashChip` for the proposal hash, and the base release (`HashChip` or `initial HEAD`).

**Progress rail**

- [ ] Four steps rendered horizontally. A step is `complete` (jade filled), `current` (jade outline), or `pending` (muted).
- [ ] Step completion is derived from **server data only**: Changes = always complete; Evidence = a current passing check exists for the proposal hash; Approval = a current approval exists; Release = `status === "RELEASED"`.

**Changes section**

- [ ] Ordered table of the proposal's Changes: ordinal, unit, action, desired fingerprint, status. Row click opens the same detail drawer used in GRF-206.

**Evidence section**

- [ ] A `Textarea` for criteria, seeded from `localStorage["gyrifi.criteria." + proposalId]`, with 3–4 preset chips (e.g. "No PII", "Claims cite a source", "Schema and payload keys unchanged", "No duplicate units").
- [ ] `Run evaluation` uses `useMutation`, shows a loading state, and renders the result in a card: pass/fail badge, `summary`, `previewFidelity`, model identity, and a list of `findings` as `{severity, unit, message}` rows with severity tones.
- [ ] When `previewFidelity === "FAST"`, show an inline note: "Preview is an overlay summary, not a simulated query result."
- [ ] When inference is disabled (from `useSystemStatus`), the evidence card states that the check was deterministic and that natural-language evaluation is off.
- [ ] If the evidence's bound hash differs from the current proposal hash, an amber banner reads: "Evidence was recorded for a different proposal hash and no longer applies." and the Approval step is forced back to `pending`.

**Approval section**

- [ ] Shows actor, timestamp, and bound hash when an approval exists.
- [ ] `Approve` is disabled with the visible reason "Run an evaluation first." when no current passing check exists.
- [ ] The approving actor is an editable field defaulting to the last used value in `localStorage`, not the hardcoded `local-user`.

**Release section**

- [ ] `Release to Qdrant` is a `danger` `Button` behind a `ConfirmDialog` stating: the target collection is mutated, `{n}` units are affected, before-images are retained, and `HEAD` will advance.
- [ ] `Release` is disabled with the visible reason "Approval required." or "Ledger HEAD moved after this proposal was created — it can no longer be released." The HEAD condition is detected by comparing `proposal.baseReleaseId` with the current HEAD **and** by surfacing the server's `409` message if it happens anyway.
- [ ] On success, the detail pane refreshes, the status becomes `RELEASED`, and a `View release` link navigates to `#releases`.
- [ ] On `503 UNAVAILABLE` ("Target apply failed; recovery is required.") the UI shows a persistent danger banner pointing at the Releases recovery surface (GRF-213), **not** a transient toast.

**Creation**

- [ ] `+ New proposal` opens a `Drawer` with a `title` `Field` and the selectable `DataTable` of `READY` Changes with explicit ordering controls and a visible note that order affects the proposal hash.
- [ ] Creation errors surface the server message verbatim.

**General**

- [ ] All five interaction states in both panes.
- [ ] Full keyboard path: list navigation with arrows, section expansion with `Enter`, dialog focus trap.
- [ ] No governance decision is computed client-side beyond rendering what the server reported.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- **This ticket needs GRF-211.** Without `GET /api/v1/ledgers/{id}/proposals/{pid}` returning checks and approvals, the progress rail and the stale-evidence banner cannot be driven by server data. Do not fake it with client-side memory of the last evaluate call.
- Hash-route extension: keep the hand-rolled router; extend `Route` parsing to `#proposals/{id}` and return `{ area, id }`. Do not add a router dependency.
- Criteria presets belong in `features/proposals/criteria-presets.ts`.
- Split the page: `proposals-page.tsx` (layout + list), `proposal-detail.tsx`, `evidence-panel.tsx`, `approval-panel.tsx`, `release-panel.tsx`, `create-proposal-drawer.tsx`. One 600-line file is not acceptable.
- Gate reasons should be a small pure function returning `{ enabled: boolean; reason?: string }` per action, unit-tested independently.

## Test plan

- `features/proposals/gates.test.ts` — every combination of (check present/absent, passing/failing, hash match/mismatch, approval present/absent, HEAD match/mismatch, status) maps to the expected enabled/reason pair.
- `features/proposals/proposal-detail.test.tsx` — progress rail states; stale-evidence banner; findings rendering; disabled reasons visible.
- `features/proposals/release-panel.test.tsx` — confirm dialog required; `503` renders a persistent banner.
- `features/proposals/create-proposal-drawer.test.tsx` — ordering controls change the submitted `changeIds` order.
- Manual: full flow against a live runtime + Qdrant.

## Docs to update

- `docs/ai/design-system.md` §5.3 — mark implemented; record the final section structure.
- `docs/ai/product.md` §6 and §7 — remove the "criteria hardcoded" gap; document the review workspace.
- `docs/ai/tech-spec.md` §11 — document the extended hash-route format.
- `docs/ai/phases/phase-1.md` — completion entry with before/after and the gate matrix.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
