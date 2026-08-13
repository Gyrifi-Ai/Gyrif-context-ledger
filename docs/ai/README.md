# Gyrifi documentation for AI agents

This folder is the authoritative, code-grounded documentation set. It is deliberately split so an agent can load **only the file it needs** instead of the whole corpus.

> Documents in `docs/archive/` are superseded. Never treat them as current. When they disagree with the code, the code wins.

## Reading map

| # | File | Load when | Approx. size |
|---|---|---|---|
| 1 | [product.md](product.md) | You need the domain model, user workflows, states, or governance invariants | ~8 KB |
| 2 | [repo-structure.md](repo-structure.md) | You need to know where a file belongs or which layer may import which | ~7 KB |
| 3 | [tech-spec.md](tech-spec.md) | You need exact API shapes, DB schema, Go/TS types, hashing, or release mechanics | ~18 KB |
| 4 | [design-system.md](design-system.md) | You are writing or reviewing Studio UI | ~16 KB |
| 5 | [tickets/INDEX.md](tickets/INDEX.md) | You are picking up implementation work | ~5 KB |
| 6 | [phases/](phases/) | You just finished a ticket, or need history of what was built and why | varies |
| 7 | [../../AGENTS.md](../../AGENTS.md) | Always, first | ~7 KB |

## Minimum context per task type

Do not load everything. Use this table.

| Task | Load |
|---|---|
| Backend feature or bug | `AGENTS.md`, `tech-spec.md`, the ticket, relevant Go files |
| Frontend feature | `AGENTS.md`, `design-system.md`, the ticket, relevant `studio/src` files |
| New endpoint | `AGENTS.md`, `tech-spec.md` (API section), `repo-structure.md` |
| Schema change | `AGENTS.md`, `tech-spec.md` (schema section), `runtime/migrations/` |
| Anything touching governance rules | `AGENTS.md`, `product.md` (invariants), `tech-spec.md` |
| Writing docs | `AGENTS.md`, the doc being changed |

## Ground truth precedence

1. Source code in `runtime/` and `studio/`
2. Tests in `runtime/tests/`, `runtime/internal/**/*_test.go`, `studio/src/**/*.test.ts`
3. `docs/ai/*.md`
4. `docs/adr/*.md`
5. `README.md`
6. `docs/archive/*` — historical only

If you find a contradiction between (1) and (3), fix (3) as part of your change and record it in the phase log.

## Conventions used in these docs

- `MUST` / `MUST NOT` are hard rules. Violating one is a rejected change.
- `SHOULD` is a strong default; deviating requires a note in the phase log.
- Code identifiers are written exactly as they appear in source.
- Anything marked **Not implemented** is a documented gap, usually with a ticket ID next to it.
