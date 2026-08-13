# GRF-233 — CI pipeline

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 4 — Qualification |
| Epic | Quality |
| Priority | Highest (do this first) |
| Size | M |
| Depends on | — |
| Blocks | Every other ticket's verification |

## Summary

Add a CI pipeline that runs the quality gate on every push and pull request. There is currently no automation of any kind — no `.github/` directory exists.

**Implement this before any other ticket.** Every subsequent ticket's definition of done says "quality gate green", and right now that is an honour system.

## Context

The quality gate documented in [tech-spec.md §13](../tech-spec.md) is:

```bash
cd runtime
go fmt ./... && go vet ./... && go test ./... && go build ./...

cd ../studio
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build

docker build -t gyrifi:ci .
```

Nothing enforces it. A commit that fails `go vet` merges silently.

## Scope

### In scope

- A GitHub Actions workflow running the gate.
- Dependency caching.
- Extension points for the integration (GRF-231) and e2e (GRF-232) jobs, added as they land.

### Out of scope

- Publishing images or releases.
- Deployment.
- Matrix builds across Go or Node versions — pin one of each and move on.

## Acceptance criteria

**Workflow**

- [ ] `.github/workflows/ci.yml` triggers on `push` to the default branch and on `pull_request`.
- [ ] `concurrency` cancels superseded runs for the same ref.
- [ ] Jobs: `runtime`, `studio`, `image`. `image` depends on the other two.
- [ ] Go version is pinned to the `go.mod` version (`1.24`) via `actions/setup-go` with `go-version-file: runtime/go.mod`.
- [ ] Node and pnpm versions are pinned; pnpm is `11.15.1` matching `package.json`.
- [ ] Caching: Go build and module cache; pnpm store keyed on `pnpm-lock.yaml`.

**Runtime job**

- [ ] Runs `go vet ./...`, `go test ./... -race`, and `go build ./...` in `runtime/`.
- [ ] Formatting is checked as a **failure**, not a fixup: `test -z "$(gofmt -l .)"` and print the offending files. `go fmt` writing files in CI is useless.
- [ ] `go mod tidy` produces no diff — a job step verifies `go.mod` and `go.sum` are clean.
- [ ] `-race` is enabled. The runtime spawns goroutines for SSE, inference supervision, and (after GRF-221) preparation; races here corrupt an audit trail.

**Studio job**

- [ ] `pnpm install --frozen-lockfile` — a lockfile mismatch fails the build rather than silently resolving.
- [ ] `pnpm typecheck`, `pnpm test`, `pnpm build`.
- [ ] Coverage thresholds enforced once GRF-230 lands.
- [ ] The workspace-path quirk (`:` in the local directory) does not apply in CI, but the scripts must still work unmodified — do **not** "fix" the scripts back to bare `vite`/`tsc` for CI. Confirm the direct-entry-point form runs correctly.

**Image job**

- [ ] `docker build` using the repository `Dockerfile` with buildx layer caching.
- [ ] Build args wired for GRF-223: `VERSION` from the tag or short SHA, `COMMIT` from `github.sha`, `BUILD_DATE` from the run timestamp.
- [ ] The built image is smoke-tested: run it, poll `/api/v1/system/status` until healthy or fail after a bounded wait, assert the reported version matches the injected one, then stop it.
- [ ] The image is exported as an artefact (or loaded into the runner's daemon) so GRF-232 can consume it without rebuilding.

**Hygiene**

- [ ] `permissions:` is set to least privilege at the workflow level (`contents: read`).
- [ ] No secrets are required for the default pipeline. If a job needs one later, it must not run on `pull_request` from forks.
- [ ] All third-party actions are pinned to a full commit SHA, not a tag — tags are mutable and this is a supply-chain surface.
- [ ] Total pipeline duration stays bounded; jobs run in parallel where they are independent.
- [ ] A `README.md` badge links to the workflow.

**Extension points**

- [ ] A commented, disabled `integration` job stub with the `qdrant/qdrant` service container, ready for GRF-231 to enable.
- [ ] A commented, disabled `e2e` job stub, ready for GRF-232 to enable.
- [ ] Both stubs include a note that they must become **required** checks when enabled.

## Implementation notes

- Keep everything in one workflow file. Splitting across files before there is a reason makes the pipeline harder to reason about.
- Run `gofmt -l` on `runtime/` only; there is no Go elsewhere.
- Use `docker/build-push-action` with `push: false` and `load: true` for the smoke test, plus `cache-from`/`cache-to` of type `gha`.
- The smoke test should use the container's own healthiness rather than a fixed sleep: poll the status endpoint in a bounded loop.
- Do not add a linter beyond `go vet` in this ticket. Introducing `golangci-lint` is a separate decision with its own configuration debate.
- Do not add commit-message or PR-title checks.

## Test plan

- Open a pull request that intentionally fails each gate in turn — unformatted Go, a `go vet` finding, a failing Go test, a type error, a failing Vitest test, a broken Dockerfile — and confirm each one fails the correct job with a legible message.
- Confirm caching works by comparing a cold and a warm run.
- Confirm the concurrency cancellation triggers on a rapid second push.

## Docs to update

- `docs/ai/tech-spec.md` §13 — the CI-enforced gate and where it runs.
- `docs/ai/repo-structure.md` — `.github/workflows/`.
- `README.md` — badge and a note that the gate is enforced.
- `AGENTS.md` — the agent must know CI enforces the gate it is asked to run locally.
- `docs/ai/phases/phase-4.md` — completion entry with the final job list and pinned versions.

## Definition of done

All acceptance criteria checked, the pipeline green on a real pull request, INDEX status updated, phase log entry written.
