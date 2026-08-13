# Phase logs

Phase logs are the project's institutional memory. Tickets say what *should* happen; phase logs record what *did*.

## Why these exist

An agent picking up ticket GRF-213 needs to know that GRF-211 extracted a shared `evaluateGates` helper, that GRF-210 added a broker whose `Publish` must be called after commit, and that GRF-212 introduced a `change_ids` snapshot column. None of that is in the tickets — tickets are written before the work. It ends up here.

Reference documentation (`product.md`, `tech-spec.md`, `repo-structure.md`, `design-system.md`) describes the system as it is **now**. Phase logs describe how it got there and why. Both are needed; neither replaces the other.

## The rule

**Every completed ticket gets an entry in its phase log, written in the same change as the implementation.**

Not afterwards. Not in a follow-up. A ticket without a log entry is not done.

## Files

| File | Phase |
|---|---|
| [phase-1.md](phase-1.md) | Studio experience — GRF-201 … GRF-208 |
| [phase-2.md](phase-2.md) | Governance API completeness — GRF-210 … GRF-214 |
| [phase-3.md](phase-3.md) | Production hardening — GRF-220 … GRF-223 |
| [phase-4.md](phase-4.md) | Qualification — GRF-230 … GRF-233 |

## Entry template

Copy this verbatim into the phase file, under `## Completed entries`, newest last.

````markdown
### GRF-NNN — <ticket title>

| | |
|---|---|
| Completed | YYYY-MM-DD |
| Commit / PR | `<sha or #123>` |
| Deviated from ticket | Yes / No |

**What was built**

<Two to five sentences. What exists now that did not before, in terms a reader who has not read the ticket can follow.>

**Files added**

- `path/to/file` — one-line purpose

**Files changed**

- `path/to/file` — what changed and why

**Files removed**

- `path/to/file` — why, and what replaced it

**Contracts introduced or changed**

<New endpoints with request/response shapes, new exported Go signatures, new TypeScript
types, new config keys, new database columns. Anything another ticket will depend on.
Paste real signatures, not descriptions of them.>

**Key decisions**

| Decision | Why | Rejected alternative | Why rejected |
|---|---|---|---|
| | | | |

**Deviations from the ticket**

<Every acceptance criterion that was not met exactly, and why. "None" is a valid answer
but must be stated explicitly. If a criterion was dropped, say who decided and on what
basis. Silent omissions are how ticket rot starts.>

**Traps for future work**

<Non-obvious constraints discovered during implementation. The things you would tell a
colleague before they touch this code. Examples: an ordering requirement, a lock that
must not be held across a call, a Qdrant behaviour that contradicted the docs, a
TypeScript inference quirk that forced a particular shape.>

**Tests added**

- `path/to/test` — what it protects against

**Docs updated**

- `docs/ai/<file>` §N — what changed

**Verification**

```
<Paste the actual output of the quality gate: go test summary, pnpm test summary,
build result. Not "all tests pass" — the output.>
```

**Follow-ups discovered**

<Work that surfaced but was correctly out of scope. If it needs a ticket, say so and
describe it well enough that the ticket can be written from this entry alone. If it was
added to INDEX.md, note the new ID.>
````

## Writing guidance

- **Write for the next agent, who has no memory of this.** Assume zero context beyond the reference docs.
- **Record what surprised you.** A decision that was obvious in hindsight is worth one line. A behaviour that contradicted your expectation is worth a paragraph.
- **Paste real signatures and real output.** Paraphrased contracts drift; pasted ones do not.
- **Be honest about deviations.** A log that always says "implemented exactly as specified" is a log nobody trusts.
- **Do not edit past entries** except to correct a factual error, and mark the correction inline. The log is append-only; the reference docs are where current truth lives.
- **Do not duplicate the reference docs.** If the entry is explaining how the system works rather than how it changed, that content belongs in `tech-spec.md` and the entry should link to it.

## Also update

Completing a ticket touches more than the phase log:

1. The affected reference docs — see the doc-update matrix in [AGENTS.md](../../../AGENTS.md).
2. The status table in [tickets/INDEX.md](../tickets/INDEX.md).
3. Repo-scoped learnings in `/memories/repo/gyrif-foundation.md`, if the ticket revealed a convention or a trap that applies beyond it.
