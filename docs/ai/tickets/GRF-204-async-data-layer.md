# GRF-204 — Async data layer with loading, error, and empty states

| Field | Value |
|---|---|
| Type | Story |
| Phase | 1 — Studio experience |
| Epic | Studio design system |
| Priority | High |
| Size | M |
| Depends on | GRF-202 |
| Blocks | GRF-205, GRF-206, GRF-207, GRF-208 |

## Summary

Introduce a tiny, dependency-free async primitive so every screen implements the mandatory interaction states from [design-system.md §6](../design-system.md). This is the state-foundation child of [GRF-240](GRF-240-mockup-led-studio-product-system.md); Runtime data remains authoritative throughout the mockup-led redesign.

## Context

Current pattern, repeated in all four feature pages:

```tsx
const [items, setItems] = useState<Ledger[]>([]);
useEffect(() => { api.ledgers().then(r => setItems(r.items)).catch(() => undefined); }, []);
```

Consequences:

- Failures are silent — the user sees a permanent empty state and cannot tell it from "no data".
- There is no loading state at all; the page flashes empty then populates.
- Mutations do not show progress, so double-submits are possible.
- `api/events.ts` exports `subscribeToEvents` which **nothing imports**.
- `studio/src/api/client.ts` throws a plain `Error`, losing the structured `code` from the API envelope.

## Scope

### In scope

- `studio/src/api/client.ts`: introduce a typed `ApiError` carrying `code`, `message`, and `status`.
- `studio/src/app/use-async.ts`: `useQuery` and `useMutation` hooks (hand-written, ~120 lines total, no dependency).
- `studio/src/ui/feedback/async-boundary.tsx`: renders the correct one of loading / error / empty / content for a query result.
- Convert all four feature pages to the new hooks (minimal change; full redesign is GRF-205…208).
- Wire `subscribeToEvents` into a `useLedgerEvents()` hook that invalidates queries — behind a flag until GRF-210 emits real events.

### Out of scope

- Adding a data-fetching library. **Not allowed** (design-system §8.1).
- Global caching or request deduplication beyond a simple in-flight guard.
- Visual redesign of the pages.

## Acceptance criteria

- [ ] `ApiError extends Error` with readonly `code: string`, `status: number`. `request<T>` throws it for every non-2xx response, preserving `body.error.code` and `body.error.message`. Falls back to `code: "UNKNOWN"` when the envelope is absent.
- [ ] `useQuery<T>(key: string, fn: (signal: AbortSignal) => Promise<T>, deps: unknown[])` returns `{ data, error, loading, refetching, refetch }`.
- [ ] `useQuery` keeps previous `data` visible during a refetch and sets `refetching: true` (design-system §6, "partial / stale").
- [ ] `useQuery` aborts the in-flight request on unmount and on dependency change; no setState-after-unmount warnings.
- [ ] `useMutation<TArgs, TResult>(fn)` returns `{ run, pending, error, result, reset }`; `run` is a no-op while `pending` (double-submit guard).
- [ ] `AsyncBoundary` props: `{ query, empty, children }`. It renders `Skeleton` while `loading && !data`, `ErrorState` (with a working `Retry` calling `refetch`) on `error`, `empty` when the resolved collection is empty, and `children(data)` otherwise.
- [ ] Content dims to `opacity: 0.6` (via a `gy-is-refetching` class) while `refetching`, and never unmounts.
- [ ] Every existing `.catch(() => undefined)` in `studio/src` is removed. `grep -rn 'catch(() => undefined)' studio/src` returns nothing.
- [ ] Every mutating action button is driven by `useMutation` and passes `loading={pending}` to `Button`.
- [ ] Mutation errors render in an `ErrorState` or a `Field` error, never only in the console.
- [ ] `useLedgerEvents(onInvalidate)` subscribes to `/events/v1` and calls back on the `ledger` event; it tolerates the current keepalive-only stream without errors or reconnect storms.
- [ ] `pnpm typecheck && pnpm test && pnpm build` pass.

## Implementation notes

Sketch:

```ts
export class ApiError extends Error {
  constructor(readonly code: string, message: string, readonly status: number) {
    super(message);
    this.name = "ApiError";
  }
}

export function useQuery<T>(key: string, fn: (signal: AbortSignal) => Promise<T>, deps: unknown[]) {
  const [state, setState] = useState<{ data?: T; error?: ApiError; loading: boolean; refetching: boolean }>(
    { loading: true, refetching: false },
  );
  const run = useCallback(() => { /* AbortController + setState */ }, deps);
  useEffect(() => run(), [run]);
  return { ...state, refetch: run };
}
```

- Keep the hooks in `app/` — they are application infrastructure, not domain (`features/`) and not visual (`ui/`).
- `key` is used only for React keys and debugging; do not build a cache. Scale does not justify it.
- Reconnect policy for `EventSource`: the browser reconnects automatically. Do **not** add a manual retry loop; just log at `debug` and let it be.
- For the empty check, `AsyncBoundary` should accept an `isEmpty?: (data: T) => boolean` with a default of `Array.isArray(data) && data.length === 0`.

## Test plan

- `app/use-async.test.tsx` — loading→success, loading→error, refetch keeps previous data, abort on unmount, mutation double-submit guard.
- `api/client.test.ts` — extend: asserts `ApiError.code === "CONFLICT"` and `status === 409` for a structured error body, and `"UNKNOWN"` for a malformed one.
- `ui/feedback/async-boundary.test.tsx` — each of the four branches.

## Docs to update

- `docs/ai/tech-spec.md` §11 — document `ApiError`, `useQuery`, `useMutation`, `AsyncBoundary`.
- `docs/ai/repo-structure.md` — add `app/use-async.ts` and `ui/feedback/async-boundary.tsx`.
- `docs/ai/phases/phase-1.md` — completion entry.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
