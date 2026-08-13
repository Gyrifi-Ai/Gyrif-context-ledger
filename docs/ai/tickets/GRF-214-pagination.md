# GRF-214 — Pagination and filtering for list endpoints

| Field | Value |
|---|---|
| Type | Story |
| Phase | 2 — Governance API completeness |
| Epic | Governance API |
| Priority | Medium |
| Size | M |
| Depends on | — |
| Blocks | — |

## Summary

Add cursor pagination and filtering to the four list endpoints. Today every list is unbounded, so a ledger with tens of thousands of Changes returns the whole table in one JSON document.

## Context

`ListChanges`, `ListProposals`, `ListReleases`, and `ListLedgers` all issue an unbounded `SELECT`. The runtime materialises every row, marshals it, and Studio renders every item. Changes accumulate fastest — they arrive from automated ingestion and are never deleted — so this is the first list that will break.

There is also no filtering. The Changes inbox (GRF-206) needs `status` and `action` filters, and it currently filters client-side over the full result set.

## Scope

### In scope

- Cursor pagination for `GET .../changes`, `.../proposals`, `.../releases`, and `/ledgers`.
- `status` and `action` filters for Changes; `status` filter for Proposals.
- A bounded default and a hard maximum.
- Studio: server-side filtering and infinite-scroll or explicit "Load more".

### Out of scope

- Offset pagination. Offsets skip and duplicate rows under concurrent inserts, which for an audit surface is unacceptable.
- Full-text search.
- Sorting options. Every list stays newest-first.

## API contract

Request:

```
GET /api/v1/ledgers/{id}/changes?limit=50&cursor=<opaque>&status=READY&action=PUT
```

Response:

```json
{ "items": [ … ], "nextCursor": "eyJ0IjoiMjAyNi0wOC0xMlQxMDowMDowMFoiLCJpIjoiY2hnXzcifQ" }
```

`nextCursor` is omitted when there are no further rows.

**Cursor encoding:** base64url of `{"t":"<createdAt RFC3339Nano>","i":"<id>"}`. Keyset predicate:

```sql
WHERE (created_at, id) < (:cursorTime, :cursorID)
ORDER BY created_at DESC, id DESC
LIMIT :limit + 1
```

The `+1` row determines whether `nextCursor` is emitted; it is not returned to the caller.

`id` is the tiebreaker because `created_at` is not unique under fast ingestion.

## Acceptance criteria

- [ ] `limit` defaults to `50`, minimum `1`, maximum `200`. A value outside the range ⇒ `400 INVALID_ARGUMENT` — do **not** silently clamp, because silent clamping makes client paging logic wrong in ways that are hard to notice.
- [ ] A malformed or undecodable `cursor` ⇒ `400 INVALID_ARGUMENT` "The cursor is not valid."
- [ ] A cursor is opaque to clients; its internal shape is not documented in the public API surface and may change.
- [ ] Results are strictly newest-first and stable: paging through a list while new rows are inserted never returns a duplicate or skips a pre-existing row.
- [ ] `status` accepts a valid enum value for the resource; anything else ⇒ `400 INVALID_ARGUMENT` naming the allowed values.
- [ ] `action` accepts `PUT` or `DELETE` on the Changes endpoint only.
- [ ] Filters combine with `AND`.
- [ ] Filters are applied in SQL, never after the fetch.
- [ ] Supporting indexes exist or are added in migration `004_list_indexes.sql`: `(ledger_id, created_at DESC, id DESC)` on `changes`, `proposals`, `releases`, and a partial or composite index covering `(ledger_id, status, created_at DESC, id DESC)` on `changes`.
- [ ] `Repository` list methods take a `ListOptions{ Limit int; Cursor *Cursor; Status *string; Action *string }` value rather than growing positional parameters.
- [ ] All SQL uses bound parameters. Filter values are never interpolated into the statement string.
- [ ] Studio `api.changes(...)` etc. accept options and return `{ items, nextCursor }`; the types in `studio/src/api/types.ts` are updated.
- [ ] Studio Changes inbox filters go to the server; the client-side `filter(...)` over the full array is removed.
- [ ] Studio lists render a `Load more` button when `nextCursor` is present and disable it while loading.
- [ ] Existing callers that omit `limit` and `cursor` keep working and receive the first 50 items — document this as a behaviour change in the phase log.
- [ ] `go test ./...`, `pnpm typecheck`, `pnpm test` pass.

## Implementation notes

- Put the cursor codec in `runtime/internal/repository/cursor.go` with `EncodeCursor(t time.Time, id string) string` and `DecodeCursor(string) (Cursor, error)`. Both are pure and unit-testable.
- Use `base64.RawURLEncoding` — no padding in URLs.
- SQLite supports row-value comparison `(a, b) < (?, ?)`; confirm against the pinned `modernc.org/sqlite` version and fall back to the expanded form `a < ? OR (a = ? AND b < ?)` if it does not plan well.
- Verify index usage with `EXPLAIN QUERY PLAN` during development and paste the output into the phase log entry.
- Timestamps are stored as text; ensure the format is lexicographically sortable and matches what the cursor encodes. If existing rows use a different precision, normalise in the codec rather than migrating data.
- Do not change the response shape from a bare array to an envelope — the endpoints already return `{ "items": [...] }`.

## Test plan

- `runtime/internal/repository/cursor_test.go` — round-trip, malformed input, empty input.
- `runtime/tests/pagination_test.go`:
  - insert 205 changes, page with `limit=50`, assert 5 pages and no duplicates or gaps,
  - insert new rows between page fetches, assert previously returned ids do not reappear and no pre-existing row is skipped,
  - rows sharing an identical `created_at` are still totally ordered by the id tiebreaker,
  - `limit=0`, `limit=201`, garbage cursor, unknown status, unknown action ⇒ 400 each,
  - `status=READY&action=DELETE` returns only matching rows.
- `studio/src/api/client.test.ts` — query string construction and `nextCursor` propagation.

## Docs to update

- `docs/ai/tech-spec.md` §3 — endpoint table gains query parameters; document the pagination contract and the 50/200 limits.
- `docs/ai/tech-spec.md` §7 and §8 — `ListOptions`, new indexes, migration 004.
- `docs/ai/product.md` §7 — remove the pagination gap row.
- `docs/ai/phases/phase-2.md` — completion entry including the `EXPLAIN QUERY PLAN` output.

## Definition of done

All acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
