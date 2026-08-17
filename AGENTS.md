# AGENTS.md

Operating manual for AI agents working in this repository. **Read this before anything else, every session.**

Gyrifi is a local-first context ledger: version control and change governance for the context that AI systems depend on. The product's value is a trustworthy audit trail. A change that makes the audit trail wrong, unreadable, or forgeable is worse than no change at all.

---

## 1. Before you write code

Do these in order. Do not skip steps because a task looks small.

1. **Read this file.** You are doing that.
2. **Read the ticket.** Work is defined in `docs/ai/tickets/GRF-NNN-*.md`. If the user asked for something without a ticket, ask whether one exists before inventing scope.
3. **Load only the docs your task needs.** See the table in §2. Loading the whole corpus wastes context you will need for code.
4. **Verify the current code state.** Docs describe the system as of the last update. The code is the system. Open the files the ticket names and confirm the "Context" section still matches reality before trusting it.
5. **Read the phase log** for the current phase (`docs/ai/phases/phase-N.md`). Prior entries record traps, contracts, and decisions that the reference docs do not carry.
6. **Check `/memories/repo/gyrif-foundation.md`** for repository-scoped facts learned in earlier sessions.

### Ground truth precedence

When sources disagree:

1. Source code in `runtime/` and `studio/`
2. Tests
3. `docs/ai/*.md`
4. `docs/adr/*.md`
5. `README.md`
6. `docs/archive/*` — **historical only, never current**

If (1) and (3) contradict each other, the doc is wrong. Fix the doc as part of your change and note it in the phase log entry.

---

## 2. Minimum context per task type

| Task | Load |
|---|---|
| Backend feature or bug | this file, `docs/ai/tech-spec.md`, the ticket, the Go files named in it |
| Frontend feature | this file, `docs/ai/design-system.md`, the ticket, the `studio/src` files named in it |
| New HTTP endpoint | this file, `docs/ai/tech-spec.md` §3–§6, `docs/ai/repo-structure.md` |
| Schema change | this file, `docs/ai/tech-spec.md` §7–§8, `runtime/migrations/` |
| Anything touching governance rules | this file, `docs/ai/product.md` (domain model + invariants), `docs/ai/tech-spec.md` |
| Writing or fixing docs | this file, the doc being changed |

---

## 3. Hard rules

These are not preferences. A change that violates one is rejected.

### Architecture

- **Layering is one-directional:** `interfaces → engine → {ledger, repository, targets, inference}`. Never the reverse, never sideways between the leaf packages.
- **`ledger/` is pure.** Standard library only. No I/O, no database, no HTTP, no clock reads that are not passed in. It holds domain types and invariants.
- **`bootstrap/` is the only composition root.** Nothing else constructs the object graph.
- **`interfaces/` contains no business rules.** Handlers parse, delegate to the engine, and serialise. A governance decision made in a handler is a bug.
- **Studio never re-derives a governance decision.** The server computes whether an action is permitted and why; the UI renders that verbatim. Client-side gate logic will drift from the server and produce a UI that lies.

### Data and migrations

- **`runtime/migrations/001_initial.sql` is frozen.** Every schema change is a new numbered file.
- **Migrations are forward-only and never edited after they land.**
- **All SQL uses bound parameters.** No string interpolation of values into statements, ever.
- **No target or network I/O inside a SQLite transaction.** Holding a write transaction across an HTTP call to Qdrant will block the entire runtime.

### The invariants

`docs/ai/product.md` §5 lists the governance invariants, each with its enforcement mechanism. Several are enforced by database constraints — for example `UNIQUE (change_id)` in `proposal_changes`, which guarantees a Change belongs to at most one Proposal.

**Do not relax a constraint to make code pass.** If a constraint is in your way, either your approach is wrong or the invariant genuinely needs to change — and that is an ADR, not a line edit.

### Dependencies

- **Do not add a dependency** unless the ticket explicitly permits it. The runtime has exactly one direct Go dependency (`modernc.org/sqlite`). Studio's stack is React + the shadcn/ui set authorised on 2026-08-17 (Tailwind CSS v4, Radix UI primitives, `lucide-react`, `class-variance-authority`, `clsx`, `tailwind-merge`, `tw-animate-css`) — see `docs/ai/design-system.md` §8. Nothing beyond that set without a ticket.
- `CGO_ENABLED=0` must keep working. No cgo, no matter how convenient.

### Secrets

- Credentials, tokens, and API keys never appear in logs, error messages, or API responses.
- Compare secrets with `crypto/subtle.ConstantTimeCompare`.
- Never store a recoverable form of a credential.

### Scope

- **Implement the ticket.** Not the ticket plus improvements you noticed.
- Do not refactor code you are not otherwise changing.
- Do not add comments, docstrings, or type annotations to untouched code.
- Do not add error handling for conditions that cannot occur. Validate at boundaries.
- Do not create an abstraction for a single call site.
- If you find unrelated work worth doing, record it under **Follow-ups discovered** in the phase log entry. Do not do it.

---

## 4. Environment

**The workspace path contains a colon** (`Side projects: work`). This breaks shell-based binary resolution for pnpm scripts. `studio/package.json` therefore invokes tool entry points directly:

```json
"dev":       "node node_modules/vite/bin/vite.js",
"build":     "node node_modules/typescript/bin/tsc -b && node node_modules/vite/bin/vite.js build",
"typecheck": "node node_modules/typescript/bin/tsc -b --pretty false",
"test":      "node node_modules/vitest/vitest.mjs run"
```

**Do not "fix" these back to bare `vite` / `tsc` / `vitest`.** They will break locally. Any new script follows the same form.

---

## 5. Quality gate

Run all of it before declaring a ticket done. Not a subset.

```bash
# Runtime
cd runtime
test -z "$(gofmt -l .)" || { gofmt -l .; echo "unformatted files"; false; }
go vet ./...
go test ./... -race
go build ./...

# Studio
cd ../studio
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build

# Image
cd ..
docker build -t gyrifi:local .

# Ticket bookkeeping — every ticket file must appear in the INDEX status log, and vice versa
cd docs/ai/tickets
diff <(ls GRF-*.md | grep -oE 'GRF-[0-9]+' | sort) \
     <(grep -oE '^\| GRF-[0-9]+ \| (Not started|In progress|Done)' INDEX.md | grep -oE 'GRF-[0-9]+' | sort) \
  && echo "tickets consistent"
```

Once GRF-233 lands, CI enforces this on every push and pull request. Running it locally first is still your job.

If a step fails for a reason unrelated to your change, say so explicitly in your summary. Do not silently skip it, and do not "fix" it by weakening the check.

---

## 6. After implementation — required, not optional

A ticket is done when the code works **and** the documentation reflects it. Do all of this in the same change.

### 6.1 Doc update matrix

| What you changed | Update |
|---|---|
| Domain model, statuses, workflow, or an invariant | `docs/ai/product.md` |
| Added, removed, or moved a package or directory | `docs/ai/repo-structure.md` |
| HTTP endpoint, wire type, Go signature, config key, or schema | `docs/ai/tech-spec.md` |
| Any Studio visual, component, or interaction | `docs/ai/design-system.md` |
| Closed a documented gap | Remove the row from `product.md` §7 and/or `tech-spec.md` §14 |
| Discovered a new gap | Add a row there, and a ticket if it warrants one |
| A decision that outlives the ticket | New ADR in `docs/adr/` |
| How a user runs or operates the product | `README.md` |

Update the **specific sections**, not the whole file. Keep the docs' existing structure and tone.

### 6.2 Ticket status

Update the status table in `docs/ai/tickets/INDEX.md`. Mark the ticket `Done` with the date.

### 6.3 Phase log entry

Append an entry to `docs/ai/phases/phase-N.md` using the template in `docs/ai/phases/README.md`. This is mandatory. It must include:

- what was built, in plain language
- files added, changed, removed
- **real signatures** for any contract another ticket will depend on
- key decisions with the rejected alternatives and why
- **every deviation from the ticket's acceptance criteria**, explicitly — "None" is valid, silence is not
- traps discovered: the things you would warn a colleague about
- tests added and what they protect
- **pasted quality gate output**, not a claim that it passed
- follow-ups discovered

Write it for an agent with no memory of this session.

### 6.4 Repository memory

If you learned something that applies beyond this ticket — a convention, an environment quirk, a recurring trap — add a short line to `/memories/repo/gyrif-foundation.md`. Keep it brief; that file is loaded often.

---

## 7. Definition of done

- [ ] Every acceptance criterion in the ticket is met, or the deviation is documented in the phase log
- [ ] Layering rules respected
- [ ] No invariant weakened, no constraint dropped
- [ ] No unrequested dependency added
- [ ] Tests added for the behaviour the ticket describes, including its failure modes
- [ ] Full quality gate green, output captured
- [ ] Reference docs updated per the matrix in §6.1
- [ ] `tickets/INDEX.md` status updated
- [ ] Phase log entry written
- [ ] Repo memory updated if applicable

---

## 8. Things that will get a change rejected

- Governance logic in an HTTP handler or in the browser
- A gate reason computed client-side instead of rendered from the server
- Dropping or relaxing a database constraint to make a test pass
- Editing `001_initial.sql` or any landed migration
- Network I/O inside a SQLite transaction
- A new dependency the ticket did not authorise
- Secrets in logs, errors, or responses
- `--no-verify`, skipped tests, or a weakened assertion used to get green
- Treating `docs/archive/` as current
- A completed ticket with no phase log entry
- Refactoring beyond the ticket's scope
- Claiming the quality gate passed without running it

---

## 9. When you are blocked

- **The ticket contradicts the code.** The code wins. Say so, describe the contradiction, and propose an adjusted approach before implementing.
- **The ticket requires a decision it does not make.** Make the smallest reasonable choice, implement it, and document it prominently under **Key decisions**. Do not stall.
- **The change needs an invariant to move.** Stop. Write an ADR. Get it reviewed.
- **A dependency ticket has not landed.** Do not simulate its API. Either implement the dependency first or report the blocker.
- **Something is destructive or hard to reverse.** Ask first. This includes dropping tables, force-pushing, deleting branches, and rewriting published history.

---

## 10. Quick reference

| | |
|---|---|
| Product and workflows | `docs/ai/product.md` |
| Where code goes | `docs/ai/repo-structure.md` |
| API, schema, types, mechanics | `docs/ai/tech-spec.md` |
| UI tokens, components, pages | `docs/ai/design-system.md` |
| Ticket list and order | `docs/ai/tickets/INDEX.md` |
| Implementation history | `docs/ai/phases/` |
| Superseded material | `docs/archive/` — historical only |
