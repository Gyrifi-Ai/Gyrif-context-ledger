# GRF-220 — Ingestion tokens and browser session auth

| Field | Value |
|---|---|
| Type | Story |
| Phase | 3 — Production hardening |
| Epic | Security |
| Priority | Highest (blocking any non-loopback deployment) |
| Size | XL |
| Depends on | — |
| Blocks | Any networked deployment |

## Summary

Add authentication and authorisation. The runtime currently has **none**: any client that can reach port 8080 can create Changes, approve Proposals, release to Qdrant, and roll back history.

> **Write an ADR before writing code.** This ticket changes the product's trust model. Record the decision in `docs/adr/0002-authentication-model.md` and get it reviewed before implementation.

## Context

`runtime/internal/interfaces/http/server.go` applies exactly two middlewares — request id and recovery. Every handler is reachable unauthenticated. The `approvals` endpoint accepts an `actor` string from the request body and records it as the approving human, with nothing binding it to an identity.

This is defensible for the current single-user loopback story and indefensible for anything else. The audit trail is the product; an unauthenticated audit trail is decorative.

Related: `GYRIFI_TARGET_API_KEY` is already handled as a secret for the outbound Qdrant call — follow the same handling discipline for inbound credentials.

## Two distinct client classes

They have different threat models and must not share a mechanism.

| Class | Who | Credential | Scope |
|---|---|---|---|
| **Ingestion** | Automated producers writing Changes | Bearer token, long-lived | One Ledger, write-Changes only |
| **Operator** | A human in Studio | Session cookie after login | Full governance on all Ledgers |

An ingestion token must **never** be able to approve or release. That separation is the whole point: the pipeline proposes, a human disposes.

## Scope

### In scope

- Ingestion tokens: create, list, revoke; scoped to a Ledger; write-Changes only.
- Operator authentication: a single local operator account with a password, and a signed session cookie.
- Authorisation middleware and per-route policy.
- Binding `approvals.actor` to the authenticated operator.
- Migration, CLI commands, and Studio login.

### Out of scope

- OIDC / SSO / multi-tenant identity. Explicitly deferred; note it in the ADR.
- Role hierarchies beyond the two classes above.
- Rate limiting (worth a follow-up ticket).

## Acceptance criteria

**Credential storage**

- [ ] Migration `005_auth.sql` adds `operators(id, username UNIQUE, password_hash, created_at)` and `ingestion_tokens(id, ledger_id, name, token_hash, prefix, created_at, last_used_at, revoked_at)`.
- [ ] **No credential is ever stored in a recoverable form.** Passwords use Argon2id (`golang.org/x/crypto/argon2`) with per-record salts. Tokens are stored as `sha256` of the raw token; the raw token is shown exactly once at creation.
- [ ] Argon2id parameters are recorded in the ADR and encoded in the stored hash string so they can be changed later.
- [ ] Token comparison uses `crypto/subtle.ConstantTimeCompare`. Password verification uses the Argon2id verifier, never `==`.
- [ ] Tokens are generated from `crypto/rand` with ≥256 bits of entropy and a readable `gyi_` prefix; the first 8 characters are stored as `prefix` for display.
- [ ] `golang.org/x/crypto` is the only new dependency permitted by this ticket. Nothing else.

**Authentication**

- [ ] `Authorization: Bearer <token>` authenticates an ingestion client.
- [ ] A signed, `HttpOnly`, `SameSite=Strict` session cookie authenticates an operator. `Secure` is set whenever the request is not on loopback.
- [ ] Sessions are signed with an HMAC key derived from a secret at `GYRIFI_SESSION_SECRET`, or generated and persisted under the data directory with `0600` permissions on first boot if unset.
- [ ] Session lifetime is 12 hours with a sliding refresh; logout invalidates server-side.
- [ ] `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`.
- [ ] Login failures are constant-time with respect to whether the username exists, and return an identical generic message.

**Authorisation**

- [ ] A route policy table maps every route to a required capability. Adding a route without a policy entry **fails a test** — the default is deny, and the test enumerates `server.routes()` to prove it.
- [ ] Ingestion tokens may call only `POST /api/v1/ledgers/{id}/changes` and `GET /api/v1/system/status`, and only for their own `ledgerId`. Every other route returns `403 PERMISSION_DENIED`.
- [ ] Operator sessions may call everything.
- [ ] `POST .../approvals` ignores any `actor` in the body and records the authenticated operator's username. The wire type drops the field.
- [ ] Unauthenticated requests to protected routes return `401 UNAUTHENTICATED` with the standard error envelope.
- [ ] Static Studio assets and `/api/v1/auth/login` are unauthenticated. `/events/v1` requires a session.

**Operations**

- [ ] `gyrifi operator create --username <u>` prompts for a password on stdin without echoing and never accepts it as a flag.
- [ ] `gyrifi token create --ledger <id> --name <n>` prints the raw token once with an explicit "this will not be shown again" warning.
- [ ] `gyrifi token revoke <id>` and `gyrifi token list` (showing prefix, name, ledger, last used, revoked).
- [ ] On first boot with no operator, the runtime logs a clear instruction to run `gyrifi operator create` and **refuses to serve protected routes** rather than defaulting to open.
- [ ] `GYRIFI_AUTH_DISABLED=true` exists for local development, logs a loud warning on every startup, and is rejected when the listen address is not loopback.

**Non-leakage**

- [ ] Tokens, password hashes, session values, and the session secret never appear in logs, error messages, or API responses. A test asserts this by scanning captured log output during a full authenticated flow.
- [ ] Error envelopes for auth failures do not distinguish "no such token" from "revoked token" from "wrong ledger".

**Studio**

- [ ] A login screen renders when `GET /api/v1/auth/me` returns `401`, using the GRF-201/202 design system.
- [ ] `401` on any request redirects to login; `403` renders a permission `ErrorState` and does not redirect.
- [ ] A token management screen under a new `#settings` area: list, create (with the one-time reveal and a copy button), revoke behind a `ConfirmDialog`.
- [ ] The approving-actor input from GRF-207 is removed and replaced with the authenticated identity.
- [ ] No credential is written to `localStorage`.

**General**

- [ ] All existing tests updated to authenticate; none disabled.
- [ ] `go test ./...`, `pnpm typecheck`, `pnpm test`, `pnpm build`, `docker build` pass.

## Implementation notes

- New package `runtime/internal/auth` holding hashing, token generation, session signing, and the policy table. It must not import `engine`.
- Middleware lives in `interfaces/http`; `auth` stays transport-agnostic.
- Resolve the principal once in middleware and put it in the request context via an unexported key type. Handlers read it through a typed accessor, never by re-parsing headers.
- Update `last_used_at` asynchronously or at most once per minute per token — do not write to SQLite on every ingestion request.
- The default-deny route test is the single most valuable test in this ticket. Write it first.
- Bump the `/data` permission expectations in the Dockerfile if the session secret file is added; the container already runs as uid 10001.

## Test plan

- `runtime/internal/auth` unit tests: Argon2id round-trip and rejection; token generation entropy and uniqueness; constant-time comparison; session signing, expiry, and tamper rejection.
- `runtime/tests/authz_test.go`:
  - every route in `routes()` has a policy entry (table-driven over the registered routes),
  - unauthenticated ⇒ 401 on each protected route,
  - ingestion token ⇒ 200 on its own ledger's changes, 403 on another ledger's changes, 403 on approvals/release/rollback,
  - operator session ⇒ allowed everywhere,
  - approval records the session identity and ignores a body-supplied actor,
  - revoked token ⇒ 401,
  - tampered session cookie ⇒ 401.
- Log-scanning test asserting no secret material is emitted.
- `GYRIFI_AUTH_DISABLED=true` with a non-loopback bind ⇒ startup error.
- Studio tests: 401 redirect, 403 error state, token reveal shown once.

## Docs to update

- `docs/adr/0002-authentication-model.md` — **new, written first.**
- `docs/ai/tech-spec.md` §2 (new env vars), §3 (auth middleware, new endpoints, 401/403 codes), §7/§8 (schema), and a new authentication section.
- `docs/ai/product.md` §1 and §7 — the trust model changes; remove the auth gap row.
- `docs/ai/repo-structure.md` — `internal/auth` and its layering rule.
- `docs/ai/design-system.md` §5 — login and settings pages.
- `README.md` — first-run operator creation and token issuance.
- `docs/ai/phases/phase-3.md` — completion entry with the ADR link and the Argon2id parameters.

## Definition of done

ADR merged, all acceptance criteria checked, quality gate green, INDEX status updated, phase log entry written.
