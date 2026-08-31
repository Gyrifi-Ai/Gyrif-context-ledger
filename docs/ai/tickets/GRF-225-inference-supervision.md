# GRF-225 — Inference process supervision

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Operations |
| Priority | Medium |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Supervise the `llama-server` child process after startup. Today it is started once, never watched again, and its output is thrown away.

## Context

`runtime/internal/inference/llamacpp.go`, `StartLlamaServer`:

```go
command := exec.CommandContext(ctx, executable, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--model", modelPath)
command.Stdout = nil
command.Stderr = nil
if err := command.Start(); err != nil { ... }
```

then it polls `GET /health` every 250 ms until a 45 s deadline and returns.

Three verified problems:

1. **No supervision after startup.** Nothing calls `command.Wait()`, so if `llama-server` is OOM-killed or crashes, the process becomes a zombie and the Go side never notices. The next evaluation issues an HTTP request to a dead port and fails with a connection error that gives the operator no clue what actually happened.
2. **All child output is discarded.** `Stdout` and `Stderr` are `nil`, so llama-server's diagnostics — including the reason it failed to load a model or ran out of memory — go nowhere. This is the single hardest class of problem to debug in this product and the runtime is deliberately blindfolded to it.
3. **No health signal.** `/api/v1/system/status` reports the provider *name* (`llamacpp` or `disabled`), never whether the process is alive.

Model-load failure at startup is handled correctly and is **not** in scope: `bootstrap.go` stats `settings.ModelPath` and returns an error before starting, so a misconfigured explicit opt-in fails fast. That is the right behaviour.

## Scope

### In scope

- Capturing and forwarding child process output into the structured logger.
- A supervisor that detects exit and restarts with backoff.
- Health state exposed through the existing status endpoint.
- Clear evaluation-time errors when inference is down.

### Out of scope

- Running multiple inference processes or load balancing between them.
- Model download, provisioning, or caching. The model is a mounted path by design.
- Changing the evaluation prompt, schema, or provider interface.
- Making evaluation fall back to a different provider.

## Acceptance criteria

**Output capture**

- [x] `command.Stdout` and `command.Stderr` are wired to a reader that forwards each line to the `slog` logger at `debug` level with a `component: "llama-server"` attribute.
- [x] `StartLlamaServer` takes a `*slog.Logger` parameter. It currently takes none — thread it through from `bootstrap`.
- [x] Lines are read continuously and never block the child; a stalled reader must not deadlock llama-server on a full pipe. Use a bounded scanner goroutine per stream and drain to completion on exit.
- [x] Extremely long lines are truncated rather than buffered without limit.
- [x] When startup fails to become ready within the 45 s deadline, the **captured output is included in the returned error or logged at error level** — this is the change that makes model-load failures diagnosable.

**Supervision**

- [x] A supervisor goroutine calls `command.Wait()` and records the exit.
- [x] An unexpected exit is logged at `error` with the exit code and the last N captured stderr lines.
- [x] The supervisor restarts the process with bounded exponential backoff (1s, 2s, 4s … capped at 60s), re-running the readiness poll after each start.
- [x] Restarts are capped: after a configurable number of consecutive failures (`GYRIFI_INFERENCE_MAX_RESTARTS`, default 5), the supervisor stops trying and marks inference permanently unhealthy until the process is restarted. A crash-looping child must not be restarted forever.
- [x] A successful readiness check resets the failure counter.
- [x] Exit caused by intentional shutdown (`Stop()` or context cancellation) is **not** treated as a failure and does not trigger a restart. This distinction must be explicit, not inferred from the exit code.
- [x] `Stop()` remains correct: `SIGTERM`, bounded wait, then kill. It must be safe to call while a restart is in flight, and must not race the supervisor. Verified under `-race`.
- [x] No goroutine or file descriptor leaks across repeated restarts, verified by a test that cycles the process several times.

**Health**

- [x] `LlamaServer` exposes `Healthy() bool` and `State() string` (`"ready" | "starting" | "restarting" | "failed" | "stopped"`).
- [x] `GET /api/v1/system/status` reports the inference state, feeding the `health.inference` field defined in GRF-224. If GRF-224 has not landed, add the field here and note the overlap.
- [x] An evaluation attempted while inference is not ready fails with `503 UNAVAILABLE` and the message "Evaluation is unavailable: the inference process is not running." — **not** a raw connection-refused error.
- [x] A `503` from this cause must not be recorded as a failed check against the Proposal. An infrastructure outage is not evidence about the content. Verify no `checks` row is written.
- [x] Studio's evidence panel (GRF-207) renders that state distinctly from a failed evaluation.

**General**

- [x] No new dependencies.
- [x] Behaviour is unchanged when `GYRIFI_EVALUATION_PROVIDER` is not `llamacpp` — no supervisor, no goroutines, no logs.
- [x] `go test ./...` passes with `-race`.

## Implementation notes

- The existing `llamacpp_test.go` tests the provider's HTTP behaviour against a fake server. Keep that. Supervision needs a different harness — use a tiny test binary (or `os.Args[0]` re-invocation with a sentinel env var) that can be told to exit after a delay, so restart behaviour is testable without llama-server.
- `exec.CommandContext` already kills the child when the context is cancelled. The supervisor must distinguish that from a spontaneous exit, or it will attempt a restart during shutdown. Check the context before deciding to restart.
- Guard shared state with a mutex; `Stop()`, the supervisor, and `Healthy()` are all called from different goroutines. This is the most likely place to introduce a race — write the `-race` test first.
- Keep the retained stderr ring buffer small and fixed-size. It exists for the error message, not for log storage.
- Do not log llama-server output at `info`. It is verbose and would drown the runtime's own logs.

## Test plan

- `runtime/internal/inference/supervisor_test.go` using a controllable fake child:
  - child exits unexpectedly ⇒ restart occurs, state passes through `restarting` to `ready`,
  - child crash-loops ⇒ restarts stop at the cap and state becomes `failed`,
  - a successful start resets the counter,
  - `Stop()` during a restart is clean and does not trigger another start,
  - context cancellation does not cause a restart,
  - repeated restart cycles leak no goroutines (compare `runtime.NumGoroutine()` before and after, after settling).
- Output capture: a child writing to stderr has those lines forwarded to a test logger; an over-long line is truncated.
- Startup timeout includes captured output in the error.
- Engine: evaluation while unhealthy ⇒ `503`, no `checks` row written.

## Docs to update

- `docs/ai/tech-spec.md` §10 (inference lifecycle, supervision, states, new config key), §3 (evaluation `503` case).
- `docs/ai/product.md` §7 — remove the inference supervision gap row.
- `docs/ai/design-system.md` §5.3 — the inference-unavailable presentation in the evidence panel.
- `docs/ai/phases/phase-3.md` — completion entry with the final backoff and cap values.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
