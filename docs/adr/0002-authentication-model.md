# ADR 0002: Authentication model

- Status: Proposed
- Date: 2026-08-31

## Context

Gyrifi's audit trail is currently writable by every caller that can reach the application listener. A caller can submit Changes, claim them in a Proposal, supply an arbitrary approval actor, release to the target, and create rollback Proposals. This is acceptable only for the present loopback-only development deployment. It cannot support a trustworthy networked deployment.

Gyrifi has two client classes with intentionally different authority:

- automated ingestion producers need long-lived, Ledger-scoped permission to submit Changes;
- a human operator in Studio needs full governance authority, including evaluation, approval, release, recovery, rollback, and credential administration.

The separation is a governance invariant: an ingestion pipeline must never approve or release its own work. The initial product remains local-first, single-operator, and single-process. OIDC, SSO, multiple operators, and a general role hierarchy would add identity-provider and policy complexity not needed to establish this boundary.

## Decision

### Principals and capabilities

The Runtime recognizes two authenticated principal types:

| Principal | Identity | Authority |
|---|---|---|
| Ingestion | Long-lived bearer token | Read system status and create Changes in exactly one bound Ledger |
| Operator | Browser session bound to the configured local operator | Full governance and credential-administration authority across all Ledgers |

Authorization is default-deny. A transport-neutral policy table in `runtime/internal/auth` maps every exact HTTP method and route pattern to one of these capabilities:

| Capability | Ingestion | Operator | Scope rule |
|---|---:|---:|---|
| `public` | Yes | Yes | No principal required |
| `status:read` | Yes | Yes | None |
| `changes:create` | Yes | Yes | Ingestion principal's Ledger ID must equal the route Ledger ID |
| `operator` | No | Yes | All Ledgers |

The public surface is limited to:

- `GET /healthz` and `GET /readyz`;
- `POST /api/v1/auth/login`;
- Studio's static files and SPA fallback.

The metrics listener remains separate and loopback-only, outside application authentication. `GET /api/v1/system/status` requires `status:read`. `POST /api/v1/ledgers/{ledgerID}/changes` requires `changes:create`. Every other API route, `GET /events/v1`, logout, identity inspection, and ingestion-token administration requires an Operator session. Ingestion credentials presented to any other route receive `403 PERMISSION_DENIED`.

Each route is registered with an explicit policy entry. A test enumerates the registered method/pattern pairs and fails if any route lacks a policy. Unknown policy values and newly added routes deny access. Missing, malformed, unknown, expired, revoked, or tampered credentials return the standard `401 UNAUTHENTICATED` envelope. A valid principal without the required capability or Ledger scope receives the standard `403 PERMISSION_DENIED` envelope.

Migration `005_auth.sql` adds these exact persistence contracts without modifying any landed migration:

```sql
CREATE TABLE operators (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE ingestion_tokens (
	id TEXT PRIMARY KEY,
	ledger_id TEXT NOT NULL REFERENCES ledgers(id),
	name TEXT NOT NULL,
	token_hash BLOB NOT NULL UNIQUE CHECK (length(token_hash) = 32),
	prefix TEXT NOT NULL,
	created_at TEXT NOT NULL,
	last_used_at TEXT,
	revoked_at TEXT
);

CREATE INDEX ingestion_tokens_active_prefix_idx
	ON ingestion_tokens(prefix) WHERE revoked_at IS NULL;
```

The partial prefix index retrieves authentication candidates without scanning token history. Operator creation checks the singleton rule inside a write transaction; the table's unique username remains defense in depth.

### Operator credential

The initial deployment permits exactly one local operator record. The CLI rejects creation of a second operator; future multi-operator support requires a new decision because it changes identity lifecycle and audit semantics. Usernames are trimmed, non-empty, stored as entered, and matched exactly.

Passwords are hashed with Argon2id from `golang.org/x/crypto/argon2` using:

| Parameter | Value |
|---|---:|
| Argon2 version | 19 |
| Memory | 19,456 KiB |
| Iterations | 2 |
| Parallelism | 1 |
| Salt | 16 random bytes |
| Derived key | 32 bytes |

The stored value uses the self-describing form:

```text
$argon2id$v=19$m=19456,t=2,p=1$<raw-base64-salt>$<raw-base64-key>
```

Every password gets a salt from `crypto/rand`. Verification parses and bounds the encoded parameters before allocating memory, derives a candidate key, and compares it with `crypto/subtle.ConstantTimeCompare`. Login for an unknown username performs the same Argon2id work against a fixed-format dummy hash and returns the same generic error as a wrong password. Hashes, passwords, and derived keys never enter logs, errors, or responses.

`gyrifi operator create --username <username>` reads the password twice from an attached terminal with echo disabled. The supported Darwin implementation uses `syscall.SYS_IOCTL` with `syscall.TIOCGETA`/`syscall.TIOCSETA`; the supported Linux implementation uses `syscall.SYS_IOCTL` with `syscall.TCGETS`/`syscall.TCSETS`. Both copy `syscall.Termios`, clear `ECHO`, and restore the original structure with `defer` on every exit path. The small files use OS build tags and only `syscall`/`unsafe` from the standard library; unsupported platforms compile a fallback that returns a clear error without reading a password. Passwords are never accepted through a flag, environment variable, command argument, or echoed fallback; a non-terminal input that cannot guarantee this property fails closed. This preserves `golang.org/x/crypto` as the ticket's only new module.

### Ingestion tokens

An ingestion token has this wire representation:

```text
gyi_<43-character raw-base64url encoding of 32 random bytes>
```

The 32 random bytes provide 256 bits of entropy. The Runtime stores only `SHA-256(full presented token)` plus the first eight display characters, which include `gyi_`. Authentication selects non-revoked candidates by the display prefix and compares the presented token digest with every candidate using `crypto/subtle.ConstantTimeCompare`; SQLite equality is not treated as credential verification. It authenticates only when exactly one digest matches. When there are no candidates it still performs one comparison against a fixed dummy digest. Prefix collisions therefore affect lookup width but cannot authenticate the wrong token, and the unique digest constraint makes multiple matches impossible for valid data.

Token creation requires Operator authority and a Ledger ID and name. The full bearer token appears exactly once in the successful CLI or API creation response with an explicit warning that it cannot be recovered. List responses contain only ID, Ledger ID, name, prefix, creation time, last-use time, and revocation time. Revocation is terminal and retains metadata for auditability.

`last_used_at` is operational metadata, not authentication authority. A process-local mutex-protected map admits at most one update per token per minute to a bounded worker channel. A full channel leaves the admission timestamp unchanged so a later request can retry; a failed repository update makes the token immediately eligible again. The bootstrap-owned worker performs writes outside request handling and stops with the process context. Failure to update that timestamp does not fail an otherwise authorized Change request.

### Operator sessions

A successful login creates a 32-byte random opaque session ID. The cookie contains no username, role, or other identity claim. Its value is:

```text
v1.<raw-base64url-session-id>.<raw-base64url-HMAC-SHA-256-signature>
```

The signature covers the exact ASCII bytes `v1.<raw-base64url-session-id>`. Credential resolution strictly parses three segments and lengths, recomputes HMAC-SHA-256, compares the signature with `crypto/subtle.ConstantTimeCompare`, decodes and hashes the session ID, and only then looks up server state. No lookup occurs for a failed signature. Active sessions live in a process-local, mutex-protected store keyed by that SHA-256 digest and contain the Operator identity, expiry, and last refresh time. Logout removes the entry immediately. Process restart intentionally logs out all sessions; multi-replica and restart-surviving sessions are outside the single-process architecture.

Sessions expire after 12 hours of inactivity. At most once every 15 minutes, a valid request extends server state and cookie expiry to 12 hours from the current time. Expired entries are rejected and removed lazily. The cookie is named `gyrifi_session` and is always `HttpOnly`, `SameSite=Strict`, and `Path=/`. `Secure` is disabled only when the request host is `localhost` or a loopback IP; it is enabled for every other host.

Requests authenticated by session whose method is not `GET`, `HEAD`, or `OPTIONS` also require an absent `Origin` header or an `Origin` matching the request scheme and host exactly. Browser requests include `Origin`; its absence is retained for non-browser CLI/testing clients that already possess a session. Gyrifi sends no permissive CORS headers. Together with strict same-site cookies and JSON request bodies, this is the initial CSRF boundary; no separate CSRF token is introduced.

### Session signing key

`GYRIFI_SESSION_SECRET`, when set, must contain at least 32 bytes. The Runtime derives the HMAC key as `SHA-256("gyrifi/session-signing/v1\x00" || secret)`; it never stores or logs the environment value.

When the variable is unset, first boot generates 32 random bytes and atomically persists them as `${GYRIFI_DATA_DIR}/session.key` with mode `0600`. Existing files must be regular, exactly 32 bytes, and not accessible to group or other users; otherwise startup fails. The persisted signing key is required key material rather than a recoverable end-user credential. It is included in backup/restore scope and never exposed through the object store, status, metrics, logs, CLI output, or API.

### Authentication and approval identity

HTTP authentication resolves at most one principal in middleware and stores it in the request context through an unexported key and typed accessor. Handlers do not parse headers or cookies. If both a session cookie and a bearer token are present, the request returns the same `401 UNAUTHENTICATED` envelope as any invalid credential rather than choosing greater authority.

`POST /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals` no longer accepts `actor`. The handler obtains the authenticated Operator username and passes that identity to the Engine. Studio removes editable/persisted approval-actor state. Audit identity is therefore derived from authentication, never caller-authored request data.

### First boot and local development

With authentication enabled and no Operator record, the Runtime starts its public health, readiness, login, and Studio surfaces but all protected routes remain closed. Startup emits a clear instruction to run `gyrifi operator create --username <username>`. Login returns a non-secret setup-required error until the Operator exists. Readiness remains governed only by database/migration/shutdown state; authentication setup does not make the recovery and setup surfaces unreachable.

`GYRIFI_AUTH_DISABLED=true` is a development-only escape hatch. It is accepted only when `GYRIFI_HTTP_ADDRESS` has an explicit loopback host. Empty-host binds such as `:8080`, wildcard addresses, and non-loopback addresses are rejected. Disabled mode logs a prominent warning on every startup and supplies the fixed local Operator identity `local-user` to protected handlers; it does not weaken route policy code.

The local Compose process binds `:8080` inside its container, which is not loopback even though the published host port is loopback-only. Compose must therefore bootstrap a real Operator and must not enable disabled mode.

### API and CLI surface

The authentication API is:

| Method | Path | Policy | Contract |
|---|---|---|---|
| POST | `/api/v1/auth/login` | public | `{username,password}`; sets session cookie, returns authenticated Operator |
| POST | `/api/v1/auth/logout` | Operator | invalidates session, clears cookie, returns 204 |
| GET | `/api/v1/auth/me` | Operator | returns `{username}` |
| GET | `/api/v1/ingestion-tokens` | Operator | returns non-secret token metadata |
| POST | `/api/v1/ingestion-tokens` | Operator | `{ledgerId,name}`; returns metadata plus the token once |
| DELETE | `/api/v1/ingestion-tokens/{tokenID}` | Operator | idempotently revokes, returns 204 |

The corresponding administration commands are:

```text
gyrifi operator create --username <username>
gyrifi token create --ledger <ledger-id> --name <name>
gyrifi token list
gyrifi token revoke <token-id>
```

CLI administration composes the same repository and authentication services as HTTP through `bootstrap`; it does not construct a parallel object graph.

## Security boundaries

This model protects governance authority from unauthenticated network callers and separates producer authority from human authority. It does not protect against:

- compromise of the host, Runtime process, Operator browser, or Gyrifi data directory;
- interception on an unencrypted network — non-loopback deployments must place the application behind TLS, and ingestion tokens must never be sent over plaintext HTTP;
- malicious Studio JavaScript executing in the authenticated origin;
- denial-of-service before GRF-226 adds rate limiting;
- multiple replicas, shared session state, external identity lifecycle, or per-Ledger Operator roles.

Target and inference credentials remain independent outbound secrets and never authenticate inbound callers.

## Consequences

### Positive

- Automated ingestion cannot approve, release, recover, roll back, or administer credentials.
- Approval actors become authenticated identities rather than caller assertions.
- Password and token database disclosure does not reveal reusable credentials directly.
- Route additions fail closed and require an explicit authorization decision.
- Sessions can be invalidated immediately without persisting bearer material in SQLite.
- Studio credentials remain in an HttpOnly cookie and never enter `localStorage`.

### Trade-offs

- Restart logs out every Operator session.
- A single local Operator and global Operator authority are intentionally coarse.
- Bearer tokens are long-lived until explicitly revoked.
- Login hashing consumes approximately 19 MiB per attempt; GRF-226 must bound abusive attempts before non-loopback deployment.
- The generated signing key becomes part of protected data-directory backup and permission management.
- Local Compose requires an explicit first-run Operator bootstrap step.

## Rejected alternatives

- **One credential type for both producers and people:** would allow an ingestion pipeline to gain governance authority or force Studio to use long-lived bearer material.
- **Caller-supplied approval actor:** produces an audit record that is not bound to identity.
- **Plain or reversibly encrypted password/token storage:** turns database or key disclosure directly into reusable credentials.
- **bcrypt or scrypt:** acceptable password hashes, but Argon2id is the ticket's selected memory-hard algorithm and supports explicit encoded tuning.
- **JWT sessions:** logout and privilege invalidation require a revocation mechanism anyway, while identity-bearing client claims increase leakage and key-rotation complexity.
- **Database-backed sessions:** would survive restarts and support replicas, but adds durable bearer-session lifecycle and write traffic that the current single-process product does not need.
- **Unsigned random session cookies:** server-side lookup would work, but signing cheaply rejects malformed/tampered values and fulfills the explicit session-signing-key contract.
- **A recoverable token lookup column containing the token:** unnecessary; prefix candidate selection plus constant-time digest comparison is sufficient.
- **Authentication-disabled Compose:** the process listens on a container wildcard address and would normalize an insecure deployment pattern.
- **OIDC/SSO, multiple operators, and role hierarchies:** deferred until deployments require external identity and finer-grained human authorization.
- **A new terminal library:** would exceed the one-new-dependency budget; small platform-specific standard-library terminal handling is sufficient.
