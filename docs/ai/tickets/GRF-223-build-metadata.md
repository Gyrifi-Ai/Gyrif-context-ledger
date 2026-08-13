# GRF-223 — Build metadata and version consistency

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 3 — Production hardening |
| Epic | Operations |
| Priority | Medium |
| Size | S |
| Depends on | — |
| Blocks | — |

## Summary

Make the runtime report one accurate version, injected at build time, everywhere it is asked.

## Context

There are two version sources and they disagree:

- `runtime/internal/bootstrap/bootstrap.go` declares `Version = "0.1.0"`, which is what `GET /api/v1/system/status` returns.
- `runtime/internal/interfaces/cli/cli.go` prints the literal string `gyrifi dev` for the `version` command.

Neither is tied to the commit that produced the binary. When an operator reports a bug, there is no way to identify which build they are running. The Docker image carries no build metadata either.

This is small, mechanical, and worth doing before anything else in Phase 3 — every subsequent bug report benefits.

## Scope

### In scope

- A single version variable set via linker flags.
- Consistent reporting from the CLI, the HTTP status endpoint, the startup log, and OCI image labels.

### Out of scope

- A release process, changelog automation, or tag conventions.

## Acceptance criteria

- [ ] A new package `runtime/internal/buildinfo` exposes:
  ```go
  var (
      Version = "dev"
      Commit  = "unknown"
      Date    = "unknown"
  )
  func String() string  // e.g. "gyrifi 0.2.0 (a1b2c3d, 2026-08-12T10:00:00Z)"
  ```
- [ ] `bootstrap.Version` is **removed**, not aliased. A grep for it returns nothing.
- [ ] The CLI `version` command prints `buildinfo.String()` and nothing else.
- [ ] `GET /api/v1/system/status` returns `{ "version", "commit", "buildDate" }` sourced from `buildinfo`.
- [ ] One startup log line reports `buildinfo.String()`.
- [ ] The Dockerfile builds with:
  ```
  -ldflags "-s -w \
    -X github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo.Version=${VERSION} \
    -X .../buildinfo.Commit=${COMMIT} \
    -X .../buildinfo.Date=${BUILD_DATE}"
  ```
  with `ARG VERSION=dev`, `ARG COMMIT=unknown`, `ARG BUILD_DATE=unknown` so a plain `docker build` still works.
- [ ] The image sets OCI labels: `org.opencontainers.image.version`, `.revision`, `.created`, `.source`, `.licenses`.
- [ ] Existing `-s -w` stripping and `CGO_ENABLED=0` are preserved. Adding `-X` must not accidentally drop them.
- [ ] A local `go build ./...` with no flags still produces a working binary reporting `dev`.
- [ ] Studio shows the version in the shell footer, sourced from `useSystemStatus`, not hardcoded.
- [ ] `docs/ai/tech-spec.md` §2 and §3 are updated in the same change.
- [ ] `go build ./... && go test ./... && docker build .` succeed.

## Implementation notes

- `buildinfo` must import nothing beyond `fmt`. It is imported by `cli`, `bootstrap`, and `interfaces/http`, so any dependency it takes on becomes universal.
- Do not read version information from `runtime/debug.ReadBuildInfo()` as the primary source — it gives module version and VCS state only when built in specific ways, and returns nothing useful for a `docker build` from a working tree. Linker flags are deterministic.
- Keep the `String()` format stable; scripts will parse it.
- If CI (GRF-233) lands first, wire the build args there in the same style.

## Test plan

- `runtime/internal/buildinfo/buildinfo_test.go` — `String()` formatting with defaults and with injected values.
- Build with explicit `-ldflags` in a test script or a `go test` that shells out, and assert the CLI output contains the injected values.
- `runtime/internal/interfaces/http` test asserting the status payload carries the buildinfo values.
- `docker build --build-arg VERSION=9.9.9` then `docker run <image> version` prints `9.9.9`.

## Docs to update

- `docs/ai/tech-spec.md` §2 (process lifecycle / CLI table), §3 (status response shape).
- `docs/ai/repo-structure.md` — new `internal/buildinfo` package.
- `README.md` — build args for a versioned image.
- `docs/ai/phases/phase-3.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
