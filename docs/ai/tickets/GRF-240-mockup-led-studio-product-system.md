# GRF-240 — Mockup-led Studio product system

| Field | Value |
|---|---|
| Type | Epic / umbrella task |
| Phase | 1 — Studio experience |
| Epic | Studio product system |
| Priority | Highest |
| Size | XL |
| Depends on | — |
| Blocks | GRF-201 … GRF-209, GRF-230, GRF-232 |

## Summary

Replace Studio's interim dark control-room treatment with the approved light, dense SaaS product system represented by the four designer mockups. The mockups define visual hierarchy, geometry, density, and interaction patterns; they do **not** introduce CRM product concepts, invented analytics, or fake collaboration into Gyrifi.

This is an umbrella work order. It coordinates the linked tickets below; it is not permission to land an unreviewable all-at-once rewrite. Each child ticket remains independently tested, documented, and logged.

## Visual contract

The reference desktop viewport is 1440 × 1024 px.

| Area | Required direction |
|---|---|
| Colour | Off-white canvas, cool light-gray navigation, white raised panels, charcoal text, restrained gray borders. Warm orange is the brand, selection, active-tab, and focus accent. Green, amber, and red remain semantic only. |
| Geometry | Fixed 248–256 px sidebar, 52–56 px utility topbar, 24–32 px content gutters, compact 36–40 px controls, and 44–48 px table rows. |
| Navigation | Brand tile, wordmark/subtitle, search with `/` shortcut, the four current workflow areas, thin icons, white active row with orange rail, optional server-backed count pills, and local-operator footer. |
| Data surfaces | Compact KPI strip, text tabs with orange underline, filter/search toolbar, semantic table, orange selection controls, and charcoal floating bulk action bar. |
| Detail surface | White right-hand drawer or overlay panel, 340–400 px wide on desktop, with compact summary, sections/tabs, durable evidence/activity content, and a pinned action area. |
| Typography | Inter/system sans for UI; 11–12 px metadata, 13–14 px body, 20–24 px titles, 28–36 px metrics; mono for IDs, hashes, fingerprints, units, and JSON. |
| Motion | 80–220 ms opacity/colour/translate changes only; immediate focus; reduced-motion support. |
| Responsive | Preserve the desktop reference at 1440 px; collapse secondary rails below 1180 px; use a navigation drawer/strip below 900 px; stack KPI/toolbars and use full-width drawers at 480 px. |

## Product translation

| Mockup pattern | Gyrifi surface |
|---|---|
| Leads dashboard | The Ledgers landing state and server-backed workflow metrics; no new dashboard route without a product decision |
| Leads table + selection bar | Changes inbox and ordered Proposal creation |
| Customer profile panel | Change, Proposal, Release-plan, and recovery detail drawers |
| Orange create/share action | Truthful create or governed workflow action only |
| CRM navigation groups | Gyrifi workflow areas: Ledgers, Changes, Proposals, Releases |

## Non-negotiable constraints

- Preserve Gyrifi's server-authoritative governance model. A browser never decides whether approval, release, recovery, or rollback is permitted.
- Render server gate reasons verbatim once GRF-211 exposes them; do not substitute locally inferred reasons.
- Do not fabricate collaborators, user avatars, sharing, trends, sparklines, activity history, billing, editable settings, or analytics without a backing contract.
- Keep the approved Studio stack. No production dependency is added by this epic.
- Required reference documentation is updated rather than deleted. It is part of Gyrifi's auditable product record.
- The CRM’s sample labels and data are visual reference only and must never appear in Studio.

## Delivery sequence

1. Re-baseline the design specification and re-scope GRF-201 … GRF-209 against this visual system.
2. GRF-201: implement light/orange tokens and global interaction styling.
3. GRF-202 + GRF-230: complete and qualify the reusable visual system.
4. GRF-204 + GRF-203 + GRF-209: data-state primitives, responsive shell, and resilience.
5. GRF-210, GRF-211, GRF-213, plus the DELETE/list response bug fix: land server contracts before dependent governance views.
6. GRF-205 … GRF-208: rebuild product workflows and their detail surfaces.
7. GRF-232: browser qualification, keyboard pass, and canonical-viewport visual calibration.

## Acceptance criteria

- [ ] [design-system.md](../design-system.md) reflects the mockup-led light/orange system and preserves accessibility, async-state, and governance constraints.
- [ ] GRF-201 … GRF-209 reference this ticket and no longer prescribe the superseded dark/jade/BEM implementation.
- [ ] Every visible number, status, action, activity item, and gate is backed by API state or is explicitly a local preference.
- [ ] All workflow pages are implemented with loading, empty, error, stale, populated, disabled, keyboard, and responsive states.
- [ ] No API dependency is simulated client-side; GRF-207 waits for GRF-211 and GRF-208 waits for GRF-213.
- [ ] Browser qualification compares populated and exceptional states at 1440, 1180, 900, and 480 px.
- [ ] All linked tickets are complete, reference docs current, phase logs contain actual quality-gate output, and the complete repository gate is green.

## Out of scope

- Authentication, user profiles, organisation/workspace sharing, multi-tenancy, billing, or collaboration.
- Server-side full-text search before GRF-214.
- Editable runtime/target configuration without explicit HTTP contracts.
- Any change to a governance invariant, migration history, or approved runtime dependency set.

## Docs to update

- `docs/ai/design-system.md` — visual source of truth and implementation status.
- `docs/ai/tickets/GRF-201-*.md` through `GRF-209-*.md` — revised visual acceptance criteria while retaining product behaviour.
- `docs/ai/repo-structure.md` — only if the Studio tree changes.
- `docs/ai/phases/phase-1.md` — completion entry after all linked work is done.

## Definition of done

This umbrella ticket completes only when every linked child ticket and required API prerequisite is complete. It carries no shortcut around the per-ticket quality gate or documentation obligations.
