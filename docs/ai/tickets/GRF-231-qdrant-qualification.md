# GRF-231 — Qdrant integration qualification

| Field | Value |
|---|---|
| Type | Chore |
| Phase | 4 — Qualification |
| Epic | Quality |
| Priority | High |
| Size | L |
| Depends on | — |
| Blocks | — |

## Summary

Verify the Qdrant adapter against a real Qdrant instance. Every existing test uses a fake, so the adapter's correctness against the actual API is currently an assumption.

## Context

`runtime/internal/targets/qdrant/qdrant_test.go` exercises the adapter against `httptest` servers returning hand-written JSON. That validates the adapter's internal logic and nothing about Qdrant.

Untested against reality:

- The exact response shape of `points/scroll`, `points/upsert`, and `points/delete` across Qdrant versions.
- Whether a `PUT` of an identical vector is genuinely a no-op.
- Vector normalisation behaviour for `Cosine` collections — Qdrant normalises server-side, so a stored vector may not byte-match what was sent. The adapter's 1e-6 comparison tolerance exists for this reason and has never been validated against the real normalisation.
- `api-key` header authentication.
- Partial failure: what Qdrant returns when some points in a batch fail.
- Behaviour when a collection does not exist, has the wrong dimension, or the wrong distance metric.

The release path's drift detection and rollback depend entirely on this adapter reading back exactly what it wrote. If that assumption is wrong, the audit trail is wrong.

## Scope

### In scope

- A build-tagged integration test package running against a real Qdrant.
- CI service container wiring.
- Coverage of the scenarios above.

### Out of scope

- Supporting additional Qdrant features (sparse vectors, named vectors, multitenancy) unless a test proves the adapter mishandles them — in which case file a follow-up, do not fix it here.
- Additional target adapters.

## Acceptance criteria

**Harness**

- [ ] Tests live in `runtime/internal/targets/qdrant/integration_test.go` behind `//go:build integration`.
- [ ] They are skipped unless `GYRIFI_TEST_QDRANT_URL` is set, so `go test ./...` on a laptop stays green.
- [ ] Each test creates and drops a uniquely named collection; tests do not share state and can run in parallel.
- [ ] Teardown runs even on failure.
- [ ] The Qdrant version under test is logged and pinned in CI.

**Correctness**

- [ ] Round-trip: upsert a vector with a payload, read it back, and assert the adapter's `Fingerprint` of the read value equals the fingerprint of the written value.
- [ ] The above is asserted for a `Cosine` collection with a **non-unit-length** input vector — this is the case where Qdrant normalises and where the 1e-6 tolerance is actually load-bearing. If the tolerance is insufficient, that is a bug to fix in this ticket.
- [ ] The same round-trip for `Dot` and `Euclid` collections, where no normalisation occurs.
- [ ] Payload key ordering differences between write and read do not change the fingerprint.
- [ ] Payload values of every JSON type (string, number, bool, null, nested object, array) round-trip identically.
- [ ] A `DELETE` of an existing point removes it; `Preview`/read then reports absence.
- [ ] A `DELETE` of an absent point is not an error.
- [ ] Re-applying an identical `PUT` leaves the fingerprint unchanged.

**Failure modes**

- [ ] A collection that does not exist produces a classified error, not a generic 500 propagated to the caller.
- [ ] A vector with the wrong dimension produces a **semantic** rejection distinguishable from a transport failure — this is the signal GRF-221 relies on to mark a Change `INVALID`.
- [ ] An unreachable Qdrant produces a retryable/unavailable classification.
- [ ] A wrong or missing `api-key` against a secured instance produces an authentication error, and the key never appears in an error message or log.
- [ ] Drift: mutate a point directly via the Qdrant API, then run the release verification path and assert the mismatch is detected and reported with the unit id.
- [ ] Partial batch failure: construct a batch where one point is invalid and assert the adapter's reported outcome matches what actually landed in Qdrant. Whatever the behaviour is, document it — silent partial application is the worst possible outcome and must at minimum be visible.

**Integration with the release path**

- [ ] At least one test drives the full `Engine` release against real Qdrant: ledger → change → proposal → evaluation → approval → release → verify, then a rollback, asserting the collection returns to its prior state.
- [ ] Before-image retention is verified by inspecting the object store after the release.

**CI**

- [ ] CI (GRF-233) runs the integration job with a pinned `qdrant/qdrant` service container and `GYRIFI_TEST_QDRANT_URL` set.
- [ ] The integration job is required, not advisory.

## Implementation notes

- Use `t.Cleanup` for collection teardown so it survives `t.Fatal`.
- Name collections `gyrifi_it_<random>` and add a best-effort sweep of stale `gyrifi_it_*` collections at suite start, in case a previous run was killed.
- Do not add a Qdrant client library. The adapter speaks REST directly and that should not change for tests.
- If the cosine test reveals that 1e-6 is too tight or too loose, change the tolerance **and** document the empirical basis in the phase log. Do not adjust it to make a test pass without understanding why.
- Keep the fake-based unit tests. They are fast and they test different things. This ticket adds a layer; it does not replace one.

## Test plan

Covered by the acceptance criteria. Additionally:

- Run the suite twice in a row against the same Qdrant instance to prove teardown is complete.
- Run with `-race`.

## Docs to update

- `docs/ai/tech-spec.md` §9 — record the verified Qdrant version, the normalisation finding, the final tolerance and its justification, and the documented partial-failure behaviour.
- `docs/ai/tech-spec.md` §12/§13 — the integration test invocation.
- `docs/ai/product.md` §7 — remove the "Qdrant only tested against fakes" gap row.
- `docs/ai/phases/phase-4.md` — completion entry with any behaviour discovered that differs from what the adapter assumed.

## Definition of done

All acceptance criteria checked, quality gate green including the integration job, INDEX status updated, phase log entry written.
