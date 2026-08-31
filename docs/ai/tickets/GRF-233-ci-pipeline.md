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

- [x] `.github/workflows/ci.yml` triggers on `push` and on `pull_request`. Push coverage includes the default branch and feature branches so the workflow can be qualified where Enterprise Managed User policy prevents PR creation.
- [x] `concurrency` cancels superseded runs for the same ref.
- [x] Jobs: `runtime`, `studio`, `image`. `image` depends on the other two.
- [x] Go version is pinned to the `go.mod` version (`1.24`) via `actions/setup-go` with `go-version-file: runtime/go.mod`.
- [x] Node and pnpm versions are pinned; pnpm is `11.15.1` matching `package.json`.
- [x] Caching: Go build and module cache; pnpm store keyed on `pnpm-lock.yaml`.

**Runtime job**

- [x] Runs `go vet ./...`, `go test ./... -race`, and `go build ./...` in `runtime/`.
- [x] Formatting is checked as a **failure**, not a fixup: `gofmt -l` prints offending files and exits non-zero.
- [x] `go mod tidy` produces no diff — a job step verifies `go.mod` and `go.sum` are clean.
- [x] `-race` is enabled. The runtime spawns goroutines for SSE, inference supervision, and (after GRF-221) preparation; races here corrupt an audit trail.

**Studio job**

- [x] `pnpm install --frozen-lockfile` — a lockfile mismatch fails the build rather than silently resolving.
- [x] `pnpm typecheck`, `pnpm test`, `pnpm build`.
- [x] Coverage thresholds enforced through GRF-230's `pnpm coverage` entry point.
- [x] The direct-entry-point scripts run unmodified in CI and locally in the colon-bearing workspace path.

**Image job**

- [x] `docker build` uses the repository `Dockerfile` through Buildx with GitHub Actions layer caching.
- [x] Build args wired for GRF-223: `VERSION` from the tag or 12-character SHA, `COMMIT` from `github.sha`, `BUILD_DATE` from the run timestamp.
- [x] The built image is smoke-tested by polling `/readyz` for at most 30 seconds, then requiring `/api/v1/system/status` to report the injected version before cleanup.
- [x] The image is loaded for smoke testing, exported as `gyrifi-ci-image`, and retained for one day.

**Hygiene**

- [x] `permissions:` is set to least privilege at the workflow level (`contents: read`).
- [x] No secrets are required for the default pipeline.
- [x] All third-party actions are pinned to a full commit SHA, not a tag.
- [x] Every job has a timeout; Runtime and Studio run in parallel before Image.
- [x] A `README.md` badge links to the workflow.

**Extension points**

- [x] A commented, disabled `integration` job stub has the pinned `qdrant/qdrant` service container ready for GRF-231.
- [x] A commented, disabled `e2e` job stub is ready for separate activation; it remains disabled as explicitly requested.
- [x] Both stubs state that they must become required checks when enabled.

## Implementation notes

- Keep everything in one workflow file. Splitting across files before there is a reason makes the pipeline harder to reason about.
- Run `gofmt -l` on `runtime/` only; there is no Go elsewhere.
- Use `docker/build-push-action` with `push: false` and `load: true` for the smoke test, plus `cache-from`/`cache-to` of type `gha`.
- The smoke test should use the container's own readiness rather than a fixed sleep: poll `/readyz` in a bounded loop.
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
