# GRF-209 — Studio resilience: error boundary, offline state, stream reconnection

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | M |
| Depends on | GRF-202, GRF-204 |
| Blocks | — |

## Summary

Make Studio fail visibly and recoverably. This is the exceptional-state child of [GRF-240](GRF-240-mockup-led-studio-product-system.md): the product system must remain truthful when rendering fails, Runtime is unreachable, or the event stream closes.

## Context

Three verified failure modes with no handling:

**No error boundary.** `studio/src/app/bootstrap.tsx` is:

```tsx
export function bootstrap(element: HTMLElement) {
  createRoot(element).render(<StrictMode><Providers><Shell /></Providers></StrictMode>);
}
```

Any component that throws unmounts the whole tree. React 19 does not render a fallback; the operator gets a white screen with no message, no request id, and no way to recover short of a reload.

**Silently swallowed stream errors.** `studio/src/api/events.ts`:

```ts
source.onerror = () => undefined;
```

`EventSource` does auto-reconnect on transport errors, but a permanent failure — the runtime stopped, a `401` after GRF-220, a proxy closing the connection — leaves the source in `CLOSED` with nothing observing it. The UI keeps showing data that is quietly going stale.

**No unreachable-runtime state.** `studio/src/api/client.ts` turns a failed `fetch` into a generic thrown error. A stopped runtime looks identical to a bug.

This matters more here than in a typical app: Studio's whole job is telling an operator the truth about governance state. Silently displaying stale or partial data is the specific failure this product exists to prevent.

## Scope

### In scope

- A root error boundary with a usable fallback.
- A section-level error boundary usable around individual pages.
- Runtime-reachability detection and a global banner.
- Stream connection state, surfaced and recoverable.

### Out of scope

- Offline-first behaviour, caching, or a service worker. Studio is an online operator tool.
- Error reporting to a remote service.
- Retrying mutations automatically — a failed approve or release must never be silently retried.

## Acceptance criteria

**Error boundaries**

- [ ] `studio/src/app/error-boundary.tsx` exports `ErrorBoundary` (a class component — React has no hook equivalent) with props `{ fallback: (error: Error, reset: () => void) => ReactNode; onError?: (error, info) => void; children }`.
- [ ] The root boundary wraps `<Shell />` inside `<Providers>` and renders a full-page fallback using the GRF-202 `ErrorState`: a plain-language message, the error message in a `CodeBlock`, a `Reload` action, and, when present, the `X-Request-ID` of the last failed request.
- [ ] A section boundary wraps each routed page so that one broken page does not take down the shell — navigation must remain usable.
- [ ] `reset` clears the boundary's error state and re-renders the subtree without a full page reload.
- [ ] The boundary logs to `console.error` exactly once per error, with the component stack.
- [ ] React `StrictMode` double-invocation does not cause the fallback to render twice or the `onError` callback to fire twice in production builds.

**Runtime reachability**

- [ ] `client.ts` distinguishes a transport failure (`fetch` rejects — runtime unreachable) from an HTTP error response (runtime reachable, request refused). `ApiError` from GRF-204 gains a discriminator, e.g. `kind: "transport" | "http"`.
- [ ] A transport failure surfaces as an app-level state, not as a per-page error.
- [ ] When the runtime is unreachable, a persistent danger banner renders below the topbar: "Cannot reach the Gyrifi runtime. Displayed data may be out of date." with a `Retry` action.
- [ ] The banner clears automatically once any request succeeds.
- [ ] While unreachable, every mutating action in the UI is disabled with that banner as the stated reason — an operator must not be able to click `Release` against a runtime that is not answering.
- [ ] Reachability polling uses `GET /api/v1/system/status` with bounded exponential backoff (1s → 30s cap), and stops polling when the tab is hidden (`document.visibilityState`).

**Stream state**

- [ ] `subscribeToEvents` is replaced with a subscription that reports state: `"connecting" | "open" | "closed"`.
- [ ] `source.onerror` is handled, not discarded. `readyState === EventSource.CLOSED` transitions the state to `closed`.
- [ ] A `closed` stream triggers explicit reconnection with bounded exponential backoff and jitter, capped, with a maximum attempt ceiling after which the UI shows a manual `Reconnect` control.
- [ ] The topbar connection indicator from GRF-203 reflects the real stream state, and is amber while reconnecting.
- [ ] On reconnect, the active view refetches — a reconnect may have missed events.
- [ ] The subscription is torn down on unmount with no leaked `EventSource` and no timer left running. Verified by a test.

**General**

- [ ] No new dependencies.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

- The error boundary must be a class component. This is the one place in Studio where that is correct; do not work around it.
- Keep reachability state in a small context provider next to the GRF-204 data layer so both the banner and the action-disabling logic read one source.
- Do **not** treat a `503 UNAVAILABLE` from the release path as "runtime unreachable". That is a *reachable* runtime reporting a target failure, and it has its own handling in GRF-208. Conflating them would send an operator to the wrong diagnosis. This distinction is the reason for the `kind` discriminator.
- `EventSource` reconnects on its own for some conditions. Do not fight it — only add explicit reconnection once `readyState` is `CLOSED`.
- Jitter matters: without it, a runtime restart causes every open tab to reconnect in lockstep.

## Test plan

- `app/error-boundary.test.tsx` — a throwing child renders the fallback; `reset` recovers; the shell survives a page-level throw.
- `api/client.test.ts` — a rejected `fetch` yields `kind: "transport"`; a `503` response yields `kind: "http"` and is not treated as unreachable.
- `app/reachability.test.tsx` — banner appears on transport failure, clears on success, disables mutating actions while shown; backoff schedule is respected with fake timers.
- `api/events.test.ts` — `CLOSED` triggers reconnection with backoff; the attempt ceiling surfaces the manual control; unmount closes the source and clears timers.

## Docs to update

- `docs/ai/design-system.md` §4 (`ErrorState` fallback usage), §6 (add the unreachable state to the mandatory interaction states).
- `docs/ai/tech-spec.md` §11 — the frontend contract for reachability and stream state.
- `docs/ai/product.md` §7 — remove the silent-failure gap row.
- `docs/ai/phases/phase-1.md` — completion entry, including the final backoff parameters.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
