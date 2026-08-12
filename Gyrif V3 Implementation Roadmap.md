# Gyrif V3 Implementation Roadmap

## Architectural translation of the V3 proposal

The implementation plan below treats **Gyrifi** as the company/brand and **gyrif** as the actual product/system. I use that terminology consistently even where the source material says “Griffy”.

The proposed V3 architecture already establishes the right product boundary: gyrif is a **local-first governance system around a mutable context store**, with three principal domain objects—**Change, Proposal/Context PR, and Release**—rather than a duplicate database, Git wrapper, distributed workflow system, or general-purpose database proxy. The backing database remains the owner of the corpus; gyrif owns the governance history, pending changes, review evidence, approvals, rollback material, and release history. fileciteturn0file0

The ticket format below is intentionally more compact than the fully expanded sample ticket while retaining the requested fields: Task ID, Task, UI, Backend/API/Storage, Rules, and UI-driven Validation. fileciteturn0file1

### Decisions that should be fixed before implementation starts

**Runtime model.** V3 should ship as a self-contained executable per supported operating system. Running:

```text
gyrif
```

starts one local process containing the backend, SQLite repository, background workers, API server and embedded frontend assets, then opens the user's normal browser. Docker, Node.js, Python, Postgres, Redis, Kafka and Temporal are not runtime prerequisites.

The browser is the UI, but the **gyrif process remains the filesystem authority**. For example, workspace selection should call the backend, which opens a native operating-system directory picker. The web File System Access specification exposes permissioned directory *handles* to web applications rather than a portable backend filesystem-path contract, so keeping workspace ownership in the native local process avoids making gyrif's core startup flow browser-dependent. citeturn4search0

**Repository model.**

```text
<workspace>/
└── .gyrif/
    ├── repository.json
    ├── state.db
    ├── objects/
    │   ├── loose/
    │   └── packs/
    └── tmp/
```

Large archives, exports and backups may use user-configured destinations rather than always living inside `.gyrif`.

**SQLite model.** Use one dedicated mutation path plus short-lived read connections. SQLite serialises writes, while WAL mode permits readers to continue against snapshots while a writer appends to the WAL. Long-running readers must therefore be avoided because they can prevent checkpoint progress and grow the WAL. citeturn0search0turn0search4

For governance writes, initialise connections explicitly with:

```text
journal_mode = WAL
synchronous = FULL
foreign_keys = ON
busy_timeout = bounded
```

In WAL mode, `synchronous=FULL` adds a WAL sync after each transaction commit. SQLite foreign-key enforcement also needs to be explicitly enabled rather than assumed. citeturn0search3turn2search0

**Change acknowledgement model.** This incorporates the user's strongest new requirement:

> A successful change-ingestion response means the complete accepted change is already durably represented locally.

The synchronous request path therefore ends at the SQLite commit. Adapter reads, fingerprints, canonicalisation, derived objects, review preparation and other enrichment belong to the asynchronous Change Engine.

A useful state progression is:

```text
HTTP request
    ↓
basic request validation
    ↓
SQLite transaction
    ├── allocate changeId
    ├── persist raw desired change
    ├── persist idempotency key
    └── commit
    ↓
202 Accepted + changeId
    ↓
background Change Engine
    ↓
READY / INVALID / RETRYING
```

A response must not be sent before the transaction has committed successfully.

**Change immutability.** An accepted Change is immutable. “Fixing” context during review creates a new corrective Change rather than rewriting an existing Change ID. This preserves the audit trail and lets Change IDs remain stable references.

**Context PR exclusivity.** A Change may be claimed by **at most one active Context PR**. This should be a database invariant, not just a UI check. A dedicated active-claim table can make `change_id` the primary/unique key, while a separate historical `proposal_changes` table retains membership history. SQLite supports unique constraints/indexes as database-level enforcement mechanisms rather than requiring application-only checks. citeturn3search2turn3search4

**Review model.** When review starts:

```text
Context PR
    +
current active ledger validations
    +
new candidate validations added to this PR
    ↓
Review Engine
    ↓
Evaluation Engine
```

Every result is bound to the exact proposed state and validation snapshot. Adding/removing a Change, adding/editing a candidate validation, or otherwise altering the reviewed inputs makes previous evaluation evidence and approvals stale.

**Release model.** Release remains the crash-sensitive boundary between two independent transactional worlds: gyrif's SQLite repository and the external context database. The Release Engine therefore uses a durable **Release Intent → apply → verify → finalise HEAD** protocol rather than pretending both systems participate in one atomic transaction. This follows the V3 proposal's core recovery model. fileciteturn0file0

**Identity scope.** V3 remains single-machine/local-first. Reviewer profiles and assignments are implemented, but they should be described honestly as local governance identities unless a later network authentication model is introduced. Strong remote multi-user collaboration is a separate architecture problem.

**Canonical hashes.** Proposal and Release governance objects should use a documented deterministic representation. RFC 8785 exists specifically to create invariant JSON suitable for repeatable hashing. It is appropriate for controlled governance metadata; adapter-specific logical values should still use adapter-defined canonical fingerprinting so arbitrary target-database semantics are not accidentally changed. citeturn1search1

### Recommended core state

The implementation should converge on approximately these tables rather than creating a separate database for each subsystem:

```text
schema_migrations
ledgers
ledger_settings
adapter_configs
actors
api_clients

changes
change_claims
proposal_changes
proposals

validations
validation_versions
proposal_validations
evaluation_runs
evaluation_results

reviewer_assignments
approvals

release_intents
release_intent_units
releases
release_changes
ledger_heads
before_images

events
object_locations
pack_registry
backup_runs
```

That preserves the V3 principle that the journal, review state and release machinery are parts of **one repository**, not separate competing histories. fileciteturn0file0

## Delivery sequence and dependency map

The safest sequence is to build the product vertically rather than implementing all backend infrastructure first.

| Delivery slice | Tickets | Demonstrable outcome |
|---|---|---|
| Runnable local product | `GRF-101`–`GRF-110` | `gyrif` opens a browser, selects/opens a workspace, creates or opens a ledger and exposes integration details |
| Durable context inbox | `GRF-111`–`GRF-118` | Applications submit durable Changes; users inspect them and construct exclusive Context PRs |
| Review and approval | `GRF-119`–`GRF-126` | Existing validations run, failures can be fixed, tests can be added, reviewers can approve exact reviewed state |
| Safe release and rollback | `GRF-127`–`GRF-132` | Approved PRs release through crash-recoverable intents, become immutable Releases and can be rolled back forward |
| Production hardening | `GRF-133`–`GRF-139` | Backup, quotas, packing, recovery, diagnostics, cross-platform packaging and failure qualification are complete |

The critical dependency chain is:

```text
GRF-101
   ↓
GRF-102
   ↓
GRF-103 → GRF-104 → GRF-105
                     ↓
                  GRF-106
                     ↓
                  GRF-107 → GRF-108
                     ↓
                  GRF-110
                     ↓
                  GRF-111 → GRF-112 → GRF-113
                                          ↓
                                       GRF-115
                                          ↓
                                GRF-117 → GRF-118
                                          ↓
                                GRF-119 → GRF-126
                                          ↓
                                GRF-127 → GRF-130
                                          ↓
                                GRF-131 → GRF-132
```

`GRF-109`, `GRF-133`–`GRF-139` can proceed partly in parallel once the repository and ledger contracts are stable.

## Foundation, startup and ledger onboarding

These tasks turn the V3 architectural model into the product's first-run experience. They incorporate the supplied workspace-selection example, the user's requirement for one-command startup, remembered workspaces, ledger configuration, database credentials, storage limits and backups. fileciteturn0file0turn0file1

**Task ID**  
`GRF-101`

**Task**  
Implement the self-contained gyrif launcher and local browser application shell so the user can start the complete product with one command and no separately installed application services.

**UI**

- Running `gyrif` opens the default browser to the local gyrif UI.
- Initial screen shows **Starting gyrif…** until backend bootstrap completes.
- Browser-open failure leaves the terminal process running and prints a safe local address the user can open manually.
- The application shell reserves locations for ledger switcher, Inbox, Context PRs, Validations, Releases and Settings.

**Backend / API / Storage**

- Compile frontend assets into the executable/distribution.
- Bind the UI/API only to loopback by default.
- Select an available local port and initialise a per-runtime UI session.
- Implement `GET /api/v1/system/status`.
- Add graceful `Ctrl+C` shutdown.
- Detect an already-running local instance; reopen it rather than starting competing application processes where possible.
- Do not require Docker or an external database.

**Rules**

- Default network exposure is localhost only.
- Starting twice must not create two processes concurrently mutating the same repository.
- Browser failure must not kill the backend.
- Application startup must not require internet access.

**Validation**

1. Install/run the packaged build and execute `gyrif`.
2. Verify the browser opens and displays the startup shell.
3. Launch `gyrif` again and verify a conflicting second writer is not created.
4. Simulate browser-launch failure and verify the terminal provides a usable local session address.
5. Stop and restart the command and verify no orphan process remains.

**Task ID**  
`GRF-102`

**Task**  
Implement repository bootstrap, SQLite schema management and separate read/write data-access paths.

**UI**

- Normally invisible after startup.
- While opening a repository, show **Preparing workspace…**.
- Migration-required state shows **Updating gyrif workspace…** with actions disabled.
- Corrupt or unsupported repositories route to a recovery/blocking screen instead of entering the ledger UI.

**Backend / API / Storage**

- Create `.gyrif/repository.json`, `state.db`, `objects/loose`, `objects/packs` and `tmp`.
- Add versioned schema migrations.
- Configure WAL, FULL synchronous mode, foreign keys and bounded busy timeouts.
- Use a single logical mutation path/short write transactions.
- Use separate short-lived read connections for UI queries.
- Add repository version and application compatibility checks.
- Never keep a SQLite transaction open while waiting on an external backing database.

SQLite WAL is designed for simultaneous readers plus a single serialised writer, while long read transactions can delay checkpoint completion; the data-access layer should explicitly reflect those properties. citeturn0search0turn0search4

**Rules**

- No UI request may directly modify the SQLite file.
- Every schema change must have an ordered migration.
- A failed migration must not leave the repository advertised as upgraded.
- A repository produced by a newer unsupported format must be blocked rather than guessed at.

**Validation**

1. Initialise a new repository through the UI.
2. Verify the UI reaches the next onboarding screen.
3. Start with an older test schema and verify migration progress followed by successful opening.
4. Inject a migration failure and verify the workspace is not opened.
5. Restart and verify repository state remains internally consistent.

**Task ID**  
`GRF-103`

**Task**  
Enable manual workspace selection, validation, initialisation and activation.

**UI**

- Entry screen: **Welcome to gyrif**.
- Primary action: **Select Workspace**.
- Backend opens the operating system's native directory picker.
- After selection show path, validation state and either **Open Workspace** or **Initialize Workspace**.
- Support cancelled picker, permission error, unsupported workspace and retry states.
- After success navigate to ledger selection.

**Backend / API / Storage**

- Add `POST /api/v1/system/select-directory`.
- Add workspace validation, initialise and activate services.
- Put gyrif-owned files only under `.gyrif/`.
- Verify directory existence and usable read/write access.
- Verify SQLite WAL operation is supported in the selected location.
- Maintain application-level active-workspace configuration separately from repository data.

SQLite WAL requires cooperating processes to reside on the same machine and is not designed for ordinary network filesystems, so V3 should reject known unsupported workspace locations or a location that fails the WAL capability check rather than weakening durability silently. citeturn0search0

**Rules**

- Do not overwrite an existing workspace.
- Do not alter unrelated files.
- Picker cancellation is not an error.
- Workspace becomes active only after validation/initialisation succeeds.
- Machine paths must not be emitted into analytics.

**Validation**

1. Select an empty writable directory and initialise it.
2. Verify the ledger-selection screen appears.
3. Select an existing valid workspace and verify **Open Workspace** appears.
4. Select a read-only/unsupported location and verify continuation is blocked with a useful message.
5. Restart and verify the workspace files remain valid and unrelated files unchanged.

**Task ID**  
`GRF-104`

**Task**  
Automatically reopen the previously active workspace when gyrif starts, with a safe fallback to workspace selection.

**UI**

- Normal returning-user startup should skip the Welcome screen.
- Briefly show **Opening your workspace…**.
- If the previous workspace is missing or inaccessible, show the Welcome screen with a non-destructive explanation and **Select Workspace**.
- Provide **Choose a different workspace** from the startup failure state.

**Backend / API / Storage**

- Persist `activeWorkspacePath`, `workspaceId` and `lastOpenedAt` in machine-level application configuration.
- Validate the remembered ID against repository metadata before activation.
- Clear or quarantine stale application config when the workspace no longer matches.
- Do not copy or migrate the workspace automatically to another directory.

**Rules**

- Never initialise a missing remembered directory automatically.
- A path that now points to a different workspace must not be silently trusted.
- Startup failure must not modify the failed workspace.
- Workspace paths remain local-only metadata.

**Validation**

1. Open a workspace and stop gyrif.
2. Restart and verify it opens automatically.
3. Rename/remove the workspace directory.
4. Restart and verify the user lands on workspace selection rather than an error loop.
5. Restore/reselect the workspace and verify normal startup resumes.

**Task ID**  
`GRF-105`

**Task**  
Implement ledger selection and the persistent main application shell.

**UI**

- After workspace activation, show **Choose a Ledger**.
- Display ledger name, backend type, health indicator and last-opened time.
- Primary actions: **Open Ledger** and **Create Ledger**.
- Main shell after opening a ledger contains Inbox, Context PRs, Validations, Releases and Settings.
- Provide a ledger switcher without changing workspace.

**Backend / API / Storage**

- Add `GET /api/v1/ledgers`.
- Add `POST /api/v1/ledgers/{id}/activate`.
- Persist last active ledger per workspace.
- Add a lightweight event stream, such as SSE, for Change Engine/evaluation/release UI updates; polling can remain the fallback.
- Opening a ledger should load local metadata even if the backing DB is temporarily unavailable.

**Rules**

- Zero ledgers routes to creation.
- A disconnected backend must not prevent reading gyrif's local audit history.
- Switching ledger must clear ledger-specific UI state.
- Operations must always carry an explicit ledger ID at the backend boundary.

**Validation**

1. Open a workspace containing multiple ledgers.
2. Switch between ledgers and verify the shell changes context.
3. Make one backing database unavailable.
4. Verify its local history remains readable while backend-dependent actions show degraded state.
5. Restart and verify the last ledger is remembered.

**Task ID**  
`GRF-106`

**Task**  
Implement the first step of the ledger-creation wizard: ledger identity and logical-context configuration.

**UI**

- **Create Ledger** wizard step **Ledger Details**.
- Inputs: name, optional description and logical-unit identification configuration required by the selected model.
- Explain that a ledger governs one context database.
- **Continue** remains disabled until valid.
- Duplicate-name and validation errors appear inline.

**Backend / API / Storage**

- Add draft-ledger creation state.
- Generate immutable `ledger_id`.
- Store name, description, creation actor and schema/format version.
- Establish empty `HEAD`.
- Do not create a backing database itself unless a future adapter explicitly supports provisioning.

**Rules**

- Ledger names are user labels; IDs are authoritative references.
- V3 supports one backing database configuration per ledger.
- A partially completed wizard must not appear as an operational ledger.
- Abandoning the wizard should remove or expire the draft safely.

**Validation**

1. Start **Create Ledger**.
2. Enter valid ledger details and proceed.
3. Trigger an invalid/duplicate name and verify inline handling.
4. Cancel midway and verify no usable incomplete ledger appears.
5. Complete creation later and verify the assigned ID persists across restart.

**Task ID**  
`GRF-107`

**Task**  
Implement the adapter contract, backend selection and connection-test step of ledger onboarding.

**UI**

- Wizard step **Context Backend**.
- Show only adapters compiled/supported by the build.
- After choosing an adapter, render its typed configuration fields.
- Provide **Test Connection** with checking/success/failure states.
- Display release guarantee: **Atomic**, **Recoverable**, or another explicitly supported capability level.
- Explain unsupported preview/release capabilities rather than hiding them.

**Backend / API / Storage**

Define a stable adapter interface around:

```text
read(unit)
fingerprint(unit)
compile(changes)
preview(proposal, mode)
apply(release_intent)
verify(release_intent)
restore(plan)
```

Expose capabilities such as:

```text
supports_atomic_apply
supports_exact_preview
supports_conditional_write
supports_batch
supports_restore
```

- Add `GET /api/v1/adapters`.
- Add connection-test endpoint.
- Persist non-secret adapter configuration separately from credentials.

This follows the adapter boundary explicitly proposed for V3 and prevents the common abstraction error of pretending every target database has identical transaction or preview behaviour. fileciteturn0file0

**Rules**

- A ledger cannot become active until mandatory adapter configuration is valid.
- Connection-test failure must not destroy entered configuration.
- Unknown capabilities default to unsupported, never optimistic.
- External calls occur outside SQLite write transactions.

**Validation**

1. Select an available adapter.
2. Enter valid configuration and verify **Connection successful**.
3. Enter an invalid endpoint or database identifier.
4. Verify the error is actionable and the wizard remains editable.
5. Save and reopen the ledger and verify adapter configuration/capabilities persist.

**Task ID**  
`GRF-108`

**Task**  
Implement secure handling of database credentials, tokens and other ledger secrets.

**UI**

- Secret inputs are masked.
- Existing saved secret displays **Configured** rather than the secret value.
- Actions: **Replace**, **Remove** and **Test Connection**.
- No secret appears in URLs, browser local storage, error details or copied diagnostics.
- Clearly distinguish ordinary configuration from sensitive values.

**Backend / API / Storage**

- Introduce `SecretStore` abstraction.
- Store only secret references in `adapter_configs`.
- Use an operating-system credential store when supported; provide a documented local protected-storage fallback suitable for the supported platforms.
- Enforce restrictive filesystem permissions/ACLs for fallback material.
- Redact secrets centrally from logs and exception serialisation.
- Never return saved secret plaintext from GET APIs.

**Rules**

- A secret can be written or replaced, not read back.
- Connection tests resolve secrets server-side.
- Export/backup behaviour for credentials must be explicit.
- Deleting a required secret marks the ledger unconfigured rather than deleting the ledger.

**Validation**

1. Save credentials during ledger setup.
2. Reopen settings and verify the original value is not exposed.
3. Replace the credential and verify the new value is used.
4. Trigger an adapter error containing the credential in a test fixture and verify logs/UI redact it.
5. Restart and verify the credential can still be resolved according to the chosen secure-storage mechanism.

**Task ID**  
`GRF-109`

**Task**  
Implement ledger storage, rollback-retention and backup configuration during onboarding and Settings.

**UI**

- Wizard step **Storage & Recovery**.
- Inputs for storage limit, rollback retention policy and backup location/schedule.
- Provide sensible defaults and an explanation of **audit history** versus **rollback payload retention**.
- Display estimated/actual storage after ledger creation.
- Allow these settings to be edited later.

**Backend / API / Storage**

- Persist `max_storage_bytes`, pressure thresholds, rollback policy and backup configuration.
- Support initial modes such as `lean`, `extended`, and `complete`.
- Validate backup destination separately from workspace.
- Record whether scheduled backups run only while gyrif is running.
- Never silently promise infinite rollback under a finite storage budget.

The V3 proposal correctly separates permanent audit history from actual retained before-value bytes because exact indefinite rollback can grow independently of the current database size. fileciteturn0file0

**Rules**

- Audit metadata is not silently deleted by rollback-retention expiry.
- Exact rollback is promised only for retained material.
- Lowering retention must show what recoverability will eventually be lost.
- Storage limits must be enforced before repository exhaustion.

**Validation**

1. Create a ledger with explicit storage/retention values.
2. Verify the settings page displays them.
3. Enter an impossible or inaccessible backup destination.
4. Verify saving is blocked without modifying the existing configuration.
5. Restart and verify saved policies remain active.

**Task ID**  
`GRF-110`

**Task**  
Provide local API integration credentials and a developer-facing integration screen so applications can submit context Changes to the selected ledger.

**UI**

- Add **Settings → Integration**.
- Show local API base address, ledger ID and example request.
- Provide **Generate API Token**, **Copy**, **Revoke**.
- Token value is shown only at creation.
- Show recently used API clients without showing credentials.

**Backend / API / Storage**

- Add API-client records with actor identity, creation time, last-used time and revocation status.
- Store only a one-way token verifier/hash where appropriate.
- Separate UI runtime session credentials from ingestion API credentials.
- Authorise API tokens to specific ledger(s).
- Add API version prefix `/api/v1`.

**Rules**

- Revoked tokens fail immediately.
- Tokens never appear in normal request logs.
- Local UI session credentials cannot be reused as long-lived ingestion credentials.
- A client cannot submit to an unauthorised ledger.

**Validation**

1. Generate a token from the Integration screen.
2. Submit a test request using the displayed example and verify authentication succeeds.
3. Revoke the token.
4. Retry and verify the API refuses it without affecting existing Changes.
5. Restart and verify revocation remains effective.

## Durable Changes and Context PR construction

This phase implements the crucial separation between **durable acceptance** and **asynchronous Change Engine work**, plus the user's explicit rule that one Change ID cannot be included simultaneously in two Context PRs. The source proposal already treats Change as the smallest stable governance unit and Proposal as an explicit selection of those Changes. fileciteturn0file0

**Task ID**  
`GRF-111`

**Task**  
Implement the durable Change-ingestion API so success is returned only after the complete accepted request is committed to SQLite.

**UI**

- No primary end-user creation screen is required; this is application-driven.
- New submissions appear quickly in Inbox as **Preparing**.
- Integration screen includes request/response example.
- Storage or validation failures can be inspected in local diagnostics.

**Backend / API / Storage**

Add:

```text
POST /api/v1/ledgers/{ledgerId}/changes
```

Minimum request:

```json
{
  "unit": "logical/unit/key",
  "action": "put",
  "value": {},
  "idempotencyKey": "client-generated-key"
}
```

- Generate `change_id` and monotonic ledger sequence.
- Persist the unprocessed desired change payload, actor, request metadata and idempotency key in one SQLite transaction.
- Return `202 Accepted` only after commit succeeds.
- Keep the synchronous path free from target DB reads, evaluation or release work.
- Set state `ACCEPTED`.

SQLite's transactional and WAL model allows this acknowledgement boundary to remain small and well-defined; `synchronous=FULL` in WAL mode adds a sync on transaction commit for durability. citeturn0search3turn0search4

**Rules**

- No committed Change means no successful response.
- V3 initially supports desired-state `PUT`/`DELETE` semantics rather than arbitrary executable patches.
- Enforce request-size/storage limits before acknowledgement.
- Basic syntactic/schema rejection happens before insertion; target-dependent validation happens later.

**Validation**

1. Submit a valid Change via the Integration example.
2. Verify the API returns a Change ID and the Inbox shows **Preparing**.
3. Inject a SQLite commit/disk failure.
4. Verify the API reports failure and no acknowledged Change appears.
5. Restart immediately after a successful response and verify the Change is still present.

**Task ID**  
`GRF-112`

**Task**  
Implement idempotent Change submission and retry semantics.

**UI**

- Duplicate retries must not create duplicate Inbox rows.
- Change detail can display the API client and accepted time, but not the raw idempotency key unless needed for local diagnostics.

**Backend / API / Storage**

- Enforce uniqueness on `(ledger_id, client_id, idempotency_key)`.
- Store a request fingerprint with the idempotency record.
- Same key + same request returns the existing `change_id`.
- Same key + different logical request returns `409 IDEMPOTENCY_KEY_REUSED`.
- Use the database constraint as the source of truth under concurrent requests.
- Do not implement an in-memory-only deduplication cache as the correctness mechanism.

SQLite UPSERT/conflict handling operates against uniqueness constraints and is suitable for making this invariant transactional. citeturn3search0

**Rules**

- One idempotency identity maps to one logical Change forever within its scope.
- Network retries must be safe.
- Concurrent duplicate requests must converge on one Change ID.
- A retry never mutates the original Change.

**Validation**

1. Send the same request five times with one idempotency key.
2. Verify every successful response returns the same Change ID.
3. Reuse that key with a different value.
4. Verify `409` and no second Change.
5. Restart and repeat the first request to verify deduplication is persistent.

**Task ID**  
`GRF-113`

**Task**  
Implement the background Change Engine that converts durably accepted Changes into review-ready Changes.

**UI**

- Inbox statuses: **Preparing**, **Ready**, **Needs attention**, **Retrying**.
- Hover/detail status explains target connectivity or validation problems.
- **Retry preparation** is available for retryable failures.
- Invalid Changes cannot be selected into a Context PR.

**Backend / API / Storage**

For each `ACCEPTED` Change:

1. Claim work persistently.
2. Validate the logical unit against the adapter.
3. Read/fingerprint current target state where required.
4. Canonicalise the desired logical value through the adapter.
5. Move larger canonical payloads into immutable object storage where applicable.
6. Persist `base_fingerprint`, `desired_fingerprint`, object reference and `READY`.
7. Clear temporary raw payload only after its durable replacement is confirmed.

- Store retry count and next-attempt time.
- Recover unfinished items by querying SQLite on startup; no external queue is required.

**Rules**

- Adapter/database outage does not lose the Change.
- Accepted is not the same as review-ready.
- External DB I/O must occur outside a long SQLite transaction.
- Permanent adapter/schema errors become `INVALID`, not infinite retries.

**Validation**

1. Submit a Change against a healthy adapter.
2. Verify it moves from **Preparing** to **Ready**.
3. Disable the backing DB and submit another Change.
4. Verify it remains safely stored and shows **Retrying/Needs attention**.
5. Restore the DB/restart gyrif and verify processing resumes without another API submission.

**Task ID**  
`GRF-114`

**Task**  
Implement the Change Inbox for browsing, filtering and selecting review-ready Changes.

**UI**

- **Inbox** is the default ledger work queue.
- Columns/cards: Change ID, logical unit, action, author/client, received time, preparation state and PR assignment.
- Filters: Ready, Preparing, Invalid, Unassigned and assigned PR.
- Support multi-select for Ready/unassigned Changes.
- Empty state explains how to send the first Change.
- Background status updates appear without full-page refresh.

**Backend / API / Storage**

- Add paginated `GET /api/v1/ledgers/{id}/changes`.
- Prefer cursor pagination based on ledger sequence rather than unbounded offset scans.
- Add indexed filters for status, sequence, unit and active claim.
- Reads use short read-only transactions/connections.
- The endpoint never performs live target DB calls just to render the inbox.

**Rules**

- Only `READY`, unreleased, non-withdrawn Changes are selectable for a new PR.
- Assigned Changes remain visible but cannot be selected elsewhere.
- Sorting is deterministic.
- UI selection does not itself reserve a Change; reservation occurs transactionally when PR creation succeeds.

**Validation**

1. Create Changes in several statuses.
2. Open Inbox and verify correct status/filter display.
3. Select several Ready Changes.
4. Cause background status changes and verify the list updates without duplicate rows.
5. Reload/restart and verify all filtering/status data is retained.

**Task ID**  
`GRF-115`

**Task**  
Implement Change detail, audit information and controlled withdrawal.

**UI**

- Clicking a Change opens a detail panel/page.
- Show logical unit, before/desired fingerprints, human-readable diff when adapter supports it, source, status and Context PR assignment.
- Unassigned pending Change offers **Withdraw Change**.
- Assigned/released Changes show why withdrawal is blocked.
- Invalid preparation shows structured error details and retry where appropriate.

**Backend / API / Storage**

- Add `GET /changes/{changeId}`.
- Add `POST /changes/{changeId}/withdraw`.
- Withdrawal records actor/time/reason; do not delete the Change.
- Preserve immutable original desired payload/hash for audit according to retention rules.
- Record lifecycle event.

**Rules**

- Released Changes cannot be withdrawn.
- Actively claimed Changes must first be removed from/cancel their PR.
- Withdrawal is an audit state, not physical deletion.
- Accepted Change contents remain immutable.

**Validation**

1. Open an unassigned Ready Change.
2. Withdraw it and verify it disappears from selectable Inbox results.
3. Attempt withdrawal of an assigned Change.
4. Verify the UI blocks it and explains the active PR.
5. Restart and verify withdrawn state and audit detail remain.

**Task ID**  
`GRF-116`

**Task**  
Create a Context PR from selected Changes while atomically claiming every selected Change ID.

**UI**

- Inbox multi-select action: **Create Context PR**.
- Show PR name/summary input and selected Change count.
- Confirmation screen lists any item that became unavailable.
- Success navigates directly to the new Context PR.
- Assigned rows subsequently show `PR-…`.

**Backend / API / Storage**

Add:

```text
POST /api/v1/ledgers/{ledgerId}/proposals
```

In one SQLite transaction:

- verify every Change is Ready and unreleased;
- create Proposal;
- write historical `proposal_changes`;
- claim every Change in `change_claims`;
- set Proposal base to current `HEAD`;
- compute initial Proposal hash;
- commit all or nothing.

`change_claims.change_id` must be unique/primary so concurrent PR creation cannot claim the same Change twice.

**Rules**

- A Change can belong to at most one active Context PR.
- If any requested Change is already claimed, do not partially create a 59-of-60 PR unless the user explicitly retries with the available set.
- Released or withdrawn Changes cannot be claimed.
- Historical membership remains after claims are later released.

**Validation**

1. Select several unassigned Changes and create a Context PR.
2. Verify all selected rows become assigned to that PR.
3. From a concurrent session attempt to create another PR with one of the same Change IDs.
4. Verify the second creation is blocked and no partial duplicate claim occurs.
5. Restart and verify claim ownership persists.

**Task ID**  
`GRF-117`

**Task**  
Implement the Context PR list, detail page and membership editing.

**UI**

- **Context PRs** lists Draft, In Review, Approved, Releasing, Released, Blocked and Cancelled PRs.
- Detail page shows title, base Release, proposal hash, selected Changes, affected units, checks, reviewers and release readiness.
- Draft/in-review PRs support **Add Changes**, **Remove**, and **Cancel PR**.
- Add dialog only shows unassigned Ready Changes.
- Cancelling shows confirmation and returns unreleased Changes to the available Inbox.

**Backend / API / Storage**

- Add proposal list/detail endpoints.
- Add transactional add/remove membership operations.
- Adding claims the Change; removal deletes active claim but preserves membership-history event.
- Cancellation releases all active claims.
- Prevent mutation once release application has begun.
- Recompute proposal identity/review staleness after membership change.

**Rules**

- Removing a Change immediately makes it eligible for another PR.
- Cancelling a PR does not withdraw its Changes.
- A Released PR is immutable.
- A `RELEASING` or recovery-required PR cannot release its claims until the release outcome is resolved.

**Validation**

1. Open a Draft PR and add an available Change.
2. Remove another Change and verify it becomes selectable elsewhere.
3. Try to add a Change already assigned to another PR.
4. Verify the action is blocked.
5. Cancel the PR, restart, and verify its unreleased Changes are unclaimed.

**Task ID**  
`GRF-118`

**Task**  
Implement Proposal hashing, effective-state reduction and user-facing PR diff.

**UI**

- Context PR header shows immutable base Release and current proposal fingerprint/hash in advanced details.
- Main view groups Changes by logical unit and shows the effective proposed outcome.
- If multiple selected Changes affect one unit, show their ordered history and final effective value.
- Show **Base moved** when ledger HEAD changes after PR creation.
- Loading/error states for adapter-assisted diff generation.

**Backend / API / Storage**

- Canonical Proposal identity contains ledger, base Release, ordered Change IDs and candidate validation versions.
- Hash deterministic canonical representation.
- Reduce selected Changes into final desired state per logical unit in ledger sequence order.
- For V3 `PUT`/`DELETE`, final state is deterministic without executing arbitrary commands.
- Cache derived diff but treat Change membership as source of truth.
- Never incorporate an unrelated pending Change merely because it affects the same unit.

RFC 8785 supplies a deterministic JSON canonicalisation mechanism suitable for controlled Proposal/Release metadata. citeturn1search1

**Rules**

- Proposal hash changes when governed Proposal content changes.
- Change order is explicit/deterministic.
- Pending Changes outside the PR do not affect evaluation.
- Hashes must reproduce across restart and implementation versions supporting the same object format.

**Validation**

1. Create a PR with several Changes, including two for one unit.
2. Verify the UI shows the correct final effective value.
3. Add/remove a Change and verify the Proposal hash changes.
4. Add an unrelated pending Change outside the PR and verify the PR diff does not change.
5. Restart and verify the same unchanged PR produces the same hash.

## Review, validation and approval

The Review Engine should not merely run ad-hoc checks attached to one PR. It should retrieve the ledger's existing validation suite, combine it with new tests proposed during review, pass that exact snapshot to the Evaluation Engine, and preserve the evidence that justified approval. That directly implements the user's observations while retaining the V3 Proposal-hash safety property. fileciteturn0file0

**Task ID**  
`GRF-119`

**Task**  
Implement a versioned ledger Validation Library.

**UI**

- Add **Validations** section.
- Display active validation name, kind, required/optional status, version, tags/applicability and last result.
- Actions: **Create Validation**, **Edit**, **Disable**.
- Editing creates a new version rather than silently replacing historical definitions.
- Empty state explains that active validations automatically participate in future Context PR reviews.

**Backend / API / Storage**

- Add `validations` and immutable `validation_versions`.
- Define validation contract: kind, versioned config, severity/gating policy and applicability metadata.
- Maintain a monotonic validation-suite revision.
- Evaluation history references exact version IDs.
- Adapter/evaluator determines executable semantics for each supported validation kind.

**Rules**

- Historical evaluation evidence must always resolve the exact validation version that ran.
- Disabled means excluded from new review snapshots, not deleted.
- Required validation failures block release.
- Unsupported validation kinds cannot be marked passed.

**Validation**

1. Create a validation from the UI.
2. Edit it and verify a new version becomes active.
3. Disable the validation.
4. Open an old evaluation and verify it still references its original version.
5. Restart and verify active/disabled state persists.

**Task ID**  
`GRF-120`

**Task**  
Implement Review Engine orchestration and creation of an immutable evaluation snapshot.

**UI**

- Context PR action: **Run Review**.
- Confirmation summarises number of Changes, affected units and active validations.
- Running state shows progress.
- Show whether evaluation uses **Fast**, **Reference**, or adapter-specific preview fidelity.
- Backend-unavailable errors keep the PR intact and allow retry.

**Backend / API / Storage**

On review start:

1. Resolve current Proposal hash.
2. Snapshot all active ledger validation version IDs.
3. Include candidate PR validation versions.
4. Record adapter/evaluator versions and preview mode.
5. Create `evaluation_run`.
6. Invoke Evaluation Engine.
7. Persist every result against the immutable run snapshot.

- Never query “whatever validations are current” while a run is already executing.
- Do not hold SQLite transactions across evaluator work.

**Rules**

- Every active required ledger validation is included.
- Candidate validations are included.
- One run's inputs never change underneath it.
- Unsupported exact-preview behaviour must be declared rather than treated as exact.

**Validation**

1. Open a PR with existing ledger validations.
2. Click **Run Review** and verify all active validations appear.
3. Disable or edit a validation while another test run is executing.
4. Verify the running evaluation keeps its original snapshot.
5. Start another run and verify the newer validation suite is used.

**Task ID**  
`GRF-121`

**Task**  
Implement evaluation-result and validation-coverage UX.

**UI**

- PR **Checks** tab summarises Passed, Failed, Error, Skipped and Running.
- Required failures are visually separated from advisory failures.
- Clicking a result shows expected/actual information, impacted logical units and evaluator diagnostics.
- Display affected-unit validation coverage when evaluator/applicability metadata supports it.
- Provide **Re-run Review** and failure-specific fix actions.

**Backend / API / Storage**

- Add evaluation-run/result read APIs.
- Store result status, duration, message, structured evidence and affected/covered unit references.
- Compute coverage only from explicit validation applicability/evaluator evidence.
- Keep result payload sizes bounded; large evidence goes to object storage.
- Stream or poll running progress.

**Rules**

- “Coverage” must not imply model-quality guarantees; it represents declared validation applicability.
- A run is successful only when all required checks pass.
- Evaluator errors are not converted to passes.
- Old runs remain inspectable.

**Validation**

1. Run a PR containing passing and failing validations.
2. Verify counts/details are correct.
3. Trigger evaluator failure rather than a validation failure.
4. Verify it appears as **Error**, not **Failed** or **Passed**.
5. Reload and verify the exact evidence remains accessible.

**Task ID**  
`GRF-122`

**Task**  
Allow reviewers to add new validation tests from inside a Context PR and carry successful additions into the ledger's future validation suite.

**UI**

- PR Checks tab includes **Add Validation**.
- User creates a candidate test with name, type, gating policy, applicability and configuration.
- Candidate is labelled **New in this PR**.
- Saving marks previous review evidence stale and offers **Run Review Again**.
- Release summary states which new validations will become part of the ledger suite.

**Backend / API / Storage**

- Create candidate immutable `validation_version`.
- Add it to `proposal_validations`.
- Include candidate validation IDs in Proposal canonical contents or equivalent governed identity.
- On successful Release finalisation, promote candidate validations into the ledger suite transactionally.
- Cancelled PR candidates remain historical but are not automatically active.

**Rules**

- A new validation must pass its configured release policy before release.
- Adding/editing candidate validation invalidates prior approvals/review readiness.
- Candidate tests cannot silently disappear from a released PR.
- Promotion happens only after successful release finalisation.

**Validation**

1. Add a candidate validation to a reviewed PR.
2. Verify the PR becomes review-stale.
3. Re-run and make the candidate pass.
4. Release using a test workflow and verify it appears in the ledger Validation Library.
5. Repeat with a cancelled PR and verify its candidate does not become active.

**Task ID**  
`GRF-123`

**Task**  
Allow users to fix a failing review by creating a corrective Change without mutating the original Change ID.

**UI**

- Failing result offers **Fix Context** when it references an editable logical unit.
- Editor starts from the PR's effective proposed value.
- Saving creates a new Change and displays **Preparing correction…**.
- Once Ready, the corrective Change is automatically added/claimed to the same PR.
- PR diff shows both original and corrective Changes but one final effective value.

**Backend / API / Storage**

- Create a normal Change through the durable Change service.
- Store optional `supersedes_change_id` or `correction_for_result_id` metadata.
- After successful asynchronous enrichment, atomically claim it for the originating PR if the PR is still editable.
- If the PR became unavailable, leave the Change safely unassigned and tell the user.
- Recompute Proposal hash.

**Rules**

- Original accepted Changes are never edited in place.
- Corrective Changes receive new Change IDs.
- Exclusivity rules apply identically.
- A correction cannot auto-attach to a Released/Cancelled/Releasing PR.

**Validation**

1. Produce a failed validation.
2. Click **Fix Context**, change the desired state and save.
3. Verify a new Change ID appears and the original still exists.
4. Cancel/lock the PR while a simulated correction is preparing.
5. Verify the completed corrective Change remains safe and unassigned rather than being attached incorrectly.

**Task ID**  
`GRF-124`

**Task**  
Implement review staleness, rerun and invalidation rules.

**UI**

- PR clearly displays **Review current** or **Review out of date**.
- Explain why it became stale: Change added/removed, correction added, validation changed, validation suite changed, adapter/evaluator contract changed where relevant.
- **Re-run Review** creates a new run while retaining earlier runs under history.
- Old results are labelled **Superseded**, not deleted.

**Backend / API / Storage**

- Calculate a `review_fingerprint` from Proposal hash, validation snapshot and relevant evaluator/adapter version inputs.
- Store it with evaluation runs and approvals.
- Current readiness requires latest successful run to match current review fingerprint.
- Never rewrite old results.
- Invalidate release readiness transactionally when governed inputs change.

**Rules**

- A stale successful run cannot authorise release.
- A new failing run cannot fall back to an older passing run.
- Historical evidence remains visible.
- Cosmetic PR fields that do not affect governed content need not invalidate review.

**Validation**

1. Run a successful review.
2. Add a Change or candidate validation.
3. Verify the PR immediately becomes stale.
4. Re-run and verify the new run becomes authoritative.
5. Reload and verify both historical and current runs are correctly labelled.

**Task ID**  
`GRF-125`

**Task**  
Implement local actor profiles, reviewer assignment and PR review ownership.

**UI**

- Settings includes **People / Reviewers**.
- Create profiles with display name and optional email/identifier.
- PR exposes **Assign reviewers** and supports assigning the current user or other configured profiles.
- Assigned reviewers and review state are visible in the PR header.
- Empty reviewer registry guides the user to add a profile.

**Backend / API / Storage**

- Add `actors` and `reviewer_assignments`.
- Associate UI session/API client with an active actor identity.
- Record assignment actor/time.
- Keep identity mechanism local-first in V3.
- Prepare interfaces so stronger authentication can replace/augment local profiles later.

**Rules**

- Assigning a reviewer is not approval.
- Deleted/deactivated profiles remain resolvable in historical records.
- The active actor must be recorded on governance actions.
- Do not represent local profiles as cryptographically verified remote identities.

**Validation**

1. Create two reviewer profiles.
2. Assign both to a PR.
3. Switch active local actor and verify attribution changes.
4. Deactivate one profile and verify old assignments remain readable.
5. Restart and verify assignments/identity records persist.

**Task ID**  
`GRF-126`

**Task**  
Implement approval policy and the final release-readiness gate.

**UI**

- PR Review section provides **Approve** / **Revoke Approval**.
- Settings supports minimum approval count and `Allow self-approval`.
- Approval displays reviewer, timestamp and exact reviewed state.
- Any invalidating edit changes approvals to **Out of date**.
- **Release** becomes enabled only when all release gates are satisfied.

**Backend / API / Storage**

- Store approvals against actor, Proposal hash and current review fingerprint.
- Evaluate readiness from:
  - current successful required review;
  - non-stale review fingerprint;
  - minimum approvals;
  - self-approval policy;
  - no unresolved conflict;
  - PR not already releasing/released;
  - all selected Changes still valid.
- Add approve/revoke endpoints.
- Readiness is computed server-side, not trusted from UI state.

**Rules**

- Approval applies only to the exact reviewed content.
- Editing governed state invalidates approval.
- User may self-approve only when ledger policy allows it.
- Reviewer assignment alone never satisfies an approval count.
- UI cannot bypass a blocked release by directly calling the Release API.

**Validation**

1. Configure an approval policy and obtain required approvals.
2. Verify **Release** becomes available after all checks pass.
3. Change PR membership.
4. Verify approvals become stale and Release is blocked.
5. Re-run/reapprove and reload to verify readiness is restored persistently.

## Release, history and rollback

This phase implements the most important correctness boundary in the system. Gyrif cannot assume that committing its SQLite state and mutating an unrelated context database happen atomically. The source proposal therefore calls for durable Release Intent state, expected fingerprints, before-images, adapter application, verification and restart reconciliation. fileciteturn0file0

**Task ID**  
`GRF-127`

**Task**  
Implement release preflight, HEAD/base checks and target-drift detection before external mutation.

**UI**

- Clicking **Release** first opens **Release Preflight**.
- Show current HEAD, PR base, review/approval status, affected units, backend guarantee and storage availability.
- Possible outcomes: **Ready to Release**, **Base moved**, **Context drift detected**, **Backend unavailable**.
- Conflicts list affected logical units and block continuation.

**Backend / API / Storage**

Preflight must:

1. acquire ledger release serialisation;
2. confirm Proposal/review fingerprints;
3. compare PR base Release with current HEAD;
4. reduce final logical desired state;
5. compile adapter operations;
6. re-read affected target fingerprints;
7. compare them with expected bases;
8. estimate/capture readiness for before-image storage.

- Do not write target context during preflight.

**Rules**

- Unexpected target state is a conflict, never a blind overwrite.
- Only one release per ledger can enter application at a time.
- Base mismatch triggers explicit rebase/conflict handling.
- Backend availability failure leaves PR unchanged.

**Validation**

1. Prepare an approved PR and run preflight.
2. Verify it reports Ready when target fingerprints match.
3. Modify one target unit externally.
4. Re-run and verify the changed unit is reported as conflict and Release is blocked.
5. Restore target state and verify readiness returns.

**Task ID**  
`GRF-128`

**Task**  
Implement the durable Release Intent state machine, before-image capture, adapter apply and verification.

**UI**

- Release page transitions through **Preparing**, **Applying**, **Verifying** and **Finalising**.
- Show affected-unit progress without pretending non-atomic adapters are atomic.
- During application, PR editing is disabled.
- Failure displays whether automatic recovery is safe or operator action is required.

**Backend / API / Storage**

Persist state such as:

```text
PREPARING
READY
APPLYING
VERIFYING
FINALISING
RELEASED
RECOVERY_REQUIRED
```

Before external mutation:

- durably store Release Intent;
- persist expected-before and desired-after fingerprints;
- store compiled operation plan;
- capture/register required before-images.

Then:

- apply idempotent/state-based adapter operations;
- verify target fingerprints;
- only proceed to Release finalisation after verification succeeds.

**Rules**

- No target mutation before durable Release Intent exists.
- Before-images required by policy must be durable before destructive mutation.
- `HEAD` must never be advanced merely because `apply()` returned success.
- Verification, not transport-level success, determines observed target outcome.

**Validation**

1. Release a passing PR normally.
2. Verify the UI passes through the documented states.
3. Inject adapter failure during apply.
4. Verify HEAD is not advanced and the intent remains inspectable/recoverable.
5. Restart and verify the unfinished intent is found rather than forgotten.

**Task ID**  
`GRF-129`

**Task**  
Implement automatic restart reconciliation and recovery UX for interrupted Releases.

**UI**

- Startup detects unfinished Releases before permitting another release for the same ledger.
- Show **Recovering interrupted release…** when outcome is safely classifiable.
- Ambiguous/external-drift case opens **Release Recovery Required** with affected units and safe actions.
- Local browsing/history remains available where safe.

**Backend / API / Storage**

For each unfinished intent compare target fingerprints against:

```text
expected-before
desired-after
unexpected-other
```

Classify:

- all before → apply did not take effect, retry may be safe;
- all after → application succeeded, proceed with verification/finalisation;
- expected mixture → adapter-specific resume/compensation;
- third-party unexpected values → stop in `RECOVERY_REQUIRED`.

- Recovery actions must themselves be persisted and idempotent.

**Rules**

- Never guess when target state contains an unknown third value.
- Do not permit a newer Release to bypass an unresolved prior intent.
- Recovery must preserve all before-images/evidence.
- Restarting repeatedly must not duplicate semantic mutations.

**Validation**

1. Kill gyrif immediately before adapter application.
2. Restart and verify recovery classifies target as unchanged.
3. Repeat with termination after target mutation but before HEAD finalisation.
4. Verify restart recognises the desired target and safely completes finalisation.
5. Introduce external drift and verify automatic recovery stops rather than overwrites it.

**Task ID**  
`GRF-130`

**Task**  
Finalise a successful immutable Release, advance ledger HEAD and close the Context PR.

**UI**

- After verification show **Release completed** and Release ID.
- PR status becomes **Released**.
- Header links to resulting Release.
- Newly promoted validations are listed.
- Inbox removes released Changes from pending views.

**Backend / API / Storage**

In one local finalisation transaction:

- create canonical immutable Release record/object;
- reference parent HEAD;
- bind Proposal/review evidence, Change IDs, before-image manifest and adapter protocol details;
- advance `ledger_heads`;
- mark selected Changes released;
- close/remove active Change claims;
- promote candidate validations;
- mark Proposal Released;
- write release event.

- Release ID derives from canonical immutable contents.

**Rules**

- Release has exactly one parent in V3.
- Historical Release contents cannot be edited.
- HEAD advances only after target verification.
- A released Change cannot later enter another PR.
- Candidate validations promote exactly once.

**Validation**

1. Complete a normal release.
2. Verify PR, Changes, validation promotions and HEAD update together.
3. Attempt to add a released Change to a new PR.
4. Verify it is blocked.
5. Restart and verify the same Release ID/HEAD remains.

**Task ID**  
`GRF-131`

**Task**  
Implement Release history, detail, diff and HEAD inspection.

**UI**

- **Releases** page displays newest first with Release ID, parent, author, time, source PR and summary.
- Current HEAD is clearly labelled.
- Release detail shows Changes, approvals, checks, validation additions, affected units and rollback-retention state.
- **Compare with parent** displays logical diff.
- Historical evidence remains read-only.

**Backend / API / Storage**

- Add Release list/detail/diff APIs.
- Build indexes by ledger, sequence, Release ID and logical unit.
- Retrieve historical payloads through object-location abstraction.
- Distinguish metadata available forever from rollback content that has expired.
- Diff work should scale with changed units rather than full database size wherever adapter semantics permit.

**Rules**

- Release history is append-only.
- Expired rollback payload does not erase audit metadata.
- Missing/corrupt historical objects surface repository-health errors, not fabricated data.
- HEAD is a mutable pointer to an immutable Release.

**Validation**

1. Create multiple Releases.
2. Open history and compare a Release with its parent.
3. Simulate an expired rollback payload in a test repository.
4. Verify audit metadata remains visible while exact rollback availability is clearly marked.
5. Restart and verify Release order/HEAD remain unchanged.

**Task ID**  
`GRF-132`

**Task**  
Implement rollback as generation of a new forward Context PR and Release rather than moving HEAD backwards.

**UI**

- Release detail provides **Restore this state** when sufficient rollback material exists.
- Preview explains affected units and that history will remain intact.
- Action creates a normal Context PR labelled **Restore to Release …**.
- The generated PR goes through evaluation, approval and release like any other.
- If required rollback material has expired, explain why exact restoration is unavailable.

**Backend / API / Storage**

- Walk before-image chains from current HEAD toward target Release.
- Compose desired logical values required to recreate target state.
- Generate ordinary corrective/restore Changes.
- Create a standard PR with `reverts_to_release_id` metadata.
- Reuse normal review, drift detection and Release Intent machinery.
- Resulting Release records the restoration relationship.

**Rules**

- Never move public HEAD backwards.
- Never rewrite old Release history.
- Restore must still detect external target conflicts.
- Rollback requires retained exact values for every affected unit needed by the plan.
- Restore does not bypass validation or approval policies.

**Validation**

1. Create several Releases affecting the same units.
2. Choose an earlier Release and click **Restore this state**.
3. Verify a normal PR containing the required restoration Changes is created.
4. Release it and verify a new Release becomes HEAD while old Releases remain unchanged.
5. Repeat where rollback material is unavailable and verify the UI blocks exact restoration honestly.

## Reliability, storage, distribution and V3 exit criteria

The remaining tasks turn the feature-complete workflow into a product that can survive disk pressure, restarts, upgrades and long-lived local history. The architectural proposal intentionally separates hot SQLite state from immutable object storage and later packing; that separation should remain invisible to normal users except where storage/recovery information is useful. fileciteturn0file0

**Task ID**  
`GRF-133`

**Task**  
Implement consistent workspace backups and verified restore.

**UI**

- **Settings → Backup & Recovery** shows destination, schedule, latest success, next run while application is running and backup size.
- Actions: **Back Up Now**, **Verify Backup**, **Restore Backup**.
- Progress and failures are visible.
- Restore requires explicit selection and confirmation and should normally restore into a new/empty workspace location.

**Backend / API / Storage**

- Use SQLite's online backup facility or an equivalent SQLite-supported snapshot mechanism for `state.db`.
- Enumerate immutable objects reachable from that database snapshot and include them in the backup bundle.
- Write backup manifest, format version and checksums.
- Verify bundle completeness before marking success.
- Restore to temporary/new location, verify, then activate.

SQLite's Online Backup API is specifically designed to produce a consistent database snapshot while allowing the live source database to continue operating. citeturn1search0turn1search2

**Rules**

- Do not simply copy a live WAL database file opportunistically.
- A failed backup does not replace the last known-good backup.
- Backup success requires manifest/object verification.
- Secret inclusion/exclusion is explicit.

**Validation**

1. Trigger **Back Up Now** while normal reads/writes occur.
2. Verify the backup reports success.
3. Restore it to a separate test location.
4. Corrupt/remove one backup object and verify **Verify Backup** fails precisely.
5. Open the valid restored workspace and verify HEAD, Changes and PR history match the snapshot.

**Task ID**  
`GRF-134`

**Task**  
Implement storage quotas, pressure states and rollback-retention enforcement.

**UI**

- Settings shows total repository use, SQLite, pending payload, rollback objects and packed archive.
- Status levels: Normal, Warning, Critical.
- Show projected/guaranteed rollback window where calculable.
- Critical state explains which actions remain allowed.
- User can adjust limits or retention.

**Backend / API / Storage**

- Track logical/physical storage categories.
- Enforce pending Change byte/count limits.
- Reserve estimated space for release before-images before Release begins.
- Run retention only against policy-eligible rollback payloads.
- Return structured `STORAGE_LIMIT_REACHED`/`INSUFFICIENT_RELEASE_SPACE`.
- Maintain minimum free-disk safety margin.

**Rules**

- Never continue accepting Changes until filesystem exhaustion.
- Never silently remove audit records.
- Never begin a Release when required before-images cannot be stored.
- Retention runs must not delete objects still reachable from protected policies/open workflows.

**Validation**

1. Set a small test storage quota.
2. Add data until Warning/Critical state appears.
3. Attempt another Change or Release beyond the safe limit.
4. Verify the action is rejected before repository integrity is endangered.
5. Increase space/retention settings and verify normal operation resumes.

**Task ID**  
`GRF-135`

**Task**  
Implement immutable object packing, object lookup and garbage collection as background maintenance.

**UI**

- No normal workflow dependency.
- Settings/Diagnostics can show loose objects, packs, reclaimable space and last maintenance run.
- Optional **Optimise Storage** action.
- Maintenance failures show warning but must not block normal history reads while old representation remains intact.

**Backend / API / Storage**

- Store objects initially in loose content-addressed form.
- Implement immutable pack format with version, per-object hash/type/length, compressed frame/index and pack checksum.
- Register pack locations in SQLite only after pack is fully written, synced and verified.
- Delete old loose copies only after registration succeeds.
- GC only unreachable objects after safety/grace period.
- Object identity must not depend on physical loose/packed representation.

This matches the V3 proposal's intended loose-object → immutable-pack lifecycle while keeping packing completely outside user-facing governance semantics. fileciteturn0file0

**Rules**

- Packing never changes Release/Proposal identity.
- At least one verified representation must exist before another is deleted.
- Partial temporary packs are disposable.
- GC may not delete open-PR, retained rollback or reachable Release objects.

**Validation**

1. Create enough test objects to trigger packing.
2. Run maintenance and verify history/content is unchanged.
3. Terminate gyrif during pack construction.
4. Restart and verify existing objects remain readable and partial pack is safely discarded/recovered.
5. Run GC and verify only unreachable eligible objects disappear.

**Task ID**  
`GRF-136`

**Task**  
Implement repository health checks, SQLite WAL health and operator-visible recovery diagnostics.

**UI**

- **Settings → Diagnostics** shows repository version, DB health, WAL size/checkpoint state, object-store health and unfinished intents.
- Actions: **Run Quick Check**, **Run Full Repository Check**, **Retry Checkpoint** where appropriate.
- Startup warning appears for serious repository-health issues.
- Copyable diagnostic bundle excludes sensitive data.

**Backend / API / Storage**

- Run SQLite `quick_check`/integrity checks as appropriate.
- Validate foreign-key relationships and referenced immutable objects.
- Track WAL size and checkpoint outcomes.
- Ensure UI/API read transactions are short enough not to pin WAL indefinitely.
- Provide `gyrif doctor`/equivalent CLI entry using the same health service.
- Rebuild derivable object-location/index state where safe.

SQLite documents that long-lived read transactions can prevent a checkpoint from completing, allowing WAL growth; this is worth treating as an explicit operational health condition rather than an invisible implementation detail. citeturn0search0

**Rules**

- Health repair cannot rewrite immutable governance history.
- Indexes may be rebuilt from canonical state; missing canonical objects cannot be invented.
- Severe corruption blocks mutating operations until resolved/backed up.
- Diagnostics must distinguish warning from correctness-threatening failure.

**Validation**

1. Run a healthy repository check and verify success.
2. Hold a synthetic long reader while generating writes.
3. Verify WAL pressure becomes visible rather than growing unnoticed.
4. Corrupt a disposable test index/object reference and run diagnostics.
5. Verify recoverable state is rebuilt or unrecoverable corruption is identified precisely.

**Task ID**  
`GRF-137`

**Task**  
Implement structured logging, audit events, secret/path redaction and support diagnostics.

**UI**

- Diagnostics exposes recent application errors by category without raw secrets.
- User can **Export Diagnostic Bundle**.
- Governance screens expose relevant actor/time events through normal history rather than raw logs.

**Backend / API / Storage**

- Structured local logging with operation/request correlation IDs.
- Add durable governance `events` sequence separately from ephemeral debug logs.
- Central redaction for credentials, API tokens and sensitive payloads.
- Avoid full workspace paths in external telemetry.
- Diagnostic bundle should include versions, error categories, adapter capabilities and safe repository metrics.
- Telemetry, if enabled at all, should be opt-in/appropriately configurable for a local-first system.

**Rules**

- Logs are never the source of truth for governance state.
- Secrets and tokens must not be logged.
- Raw context values should not enter normal telemetry.
- Local audit events must remain available without internet connectivity.

**Validation**

1. Trigger representative workspace, adapter and evaluation errors.
2. Inspect local diagnostics.
3. Verify useful correlation/error information is present.
4. Search exported diagnostics for configured test secrets/paths and verify protected values are absent/redacted.
5. Restart and verify durable governance events remain while ephemeral logs behave according to retention.

**Task ID**  
`GRF-138`

**Task**  
Ship cross-platform self-contained builds, single-command startup and safe repository/application upgrades.

**UI**

- About/Settings displays gyrif version and repository format version.
- When an application update requires repository migration, startup explains the operation before entering normal UI.
- Unsupported future repository formats display **This workspace requires a newer version of gyrif**.
- Upgrade failure provides recovery/backup guidance rather than repeatedly retrying destructively.

**Backend / API / Storage**

- Produce signed/versioned builds for supported OS/architectures.
- Bundle frontend assets, SQLite library/runtime and required native picker/credential-store components.
- No runtime dependency on Docker, Node, Python or a separately installed DB.
- Run compatibility check before migration.
- Create/verify an appropriate pre-migration backup or recovery point for destructive migrations.
- Migrations remain ordered and restart-safe.

**Rules**

- `gyrif` is enough to start an installed product.
- A binary upgrade cannot silently open an incompatible repository.
- Failed migration cannot mark itself successful.
- Upgrade logic cannot contact external services as a prerequisite for opening a local workspace unless the user explicitly opted into that dependency.

**Validation**

1. Install a clean packaged build on each supported platform.
2. Run `gyrif` and complete onboarding without installing Docker or another database.
3. Upgrade a test workspace from the previous supported schema.
4. Inject migration failure and verify safe blocked/recoverable behaviour.
5. Restart the upgraded version and verify all existing Release IDs/governance history remain unchanged.

**Task ID**  
`GRF-139`

**Task**  
Build the failure-injection, crash-consistency and end-to-end qualification suite required before V3 release.

**UI**

- No customer-facing feature beyond the correctness of all prior states.
- Test builds may expose failpoint controls under development-only diagnostics.

**Backend / API / Storage**

Automate failpoints around:

```text
Change SQLite commit
Change Engine objectisation
PR claim transaction
evaluation persistence
before-image write
Release Intent commit
adapter apply
adapter verify
Release finalisation
pack write/register/delete
backup creation
schema migration
```

Include tests for:

- duplicate API requests;
- concurrent PR claims;
- release from same HEAD;
- external DB drift;
- repeated restart recovery;
- disk full;
- WAL pressure;
- missing/corrupt objects;
- rollback across long random histories;
- backup/restore.

**Rules**

The key invariants must hold under process termination:

```text
An acknowledged Change is present after restart.

A Change cannot be actively claimed by two PRs.

A stale review/approval cannot release.

HEAD never advertises a known-incomplete target application.

An unfinished external apply remains recoverable/visible.

Packing never destroys the final valid copy of a reachable object.

Rollback never rewrites old Releases.
```

These are the practical proof obligations already implicit in the V3 architecture rather than optional QA polish. fileciteturn0file0

**Validation**

1. Run the automated failpoint matrix repeatedly over representative end-to-end workflows.
2. Restart after every injected termination and assert repository invariants.
3. Introduce concurrency, disk pressure and target drift.
4. Verify every failure ends in either a correct completed state or an explicit recoverable/blocked state—never silent corruption.
5. Run the complete browser-driven journey from clean install through workspace → ledger → Change → Context PR → evaluation → approval → Release → restart → rollback.

### V3 release gate

The system should not be considered V3-complete merely because every individual screen exists. The release candidate should demonstrate the following uninterrupted workflow:

```text
Install gyrif
    ↓
run one command
    ↓
browser opens
    ↓
select/init workspace
    ↓
create ledger
    ↓
select/configure backing DB
    ↓
configure credentials + storage + backup
    ↓
application submits Change
    ↓
HTTP success means Change is already durable
    ↓
Change Engine asynchronously prepares it
    ↓
Inbox shows Ready Change
    ↓
select Changes
    ↓
create Context PR
    ↓
those Change IDs become exclusively claimed
    ↓
Review Engine snapshots existing ledger validations
    ↓
Evaluation Engine runs them
    ↓
user sees failures + coverage
    ↓
user creates corrective Changes and/or new validations
    ↓
review reruns
    ↓
assign reviewers
    ↓
approve exact reviewed state
    ↓
release preflight verifies HEAD + target fingerprints
    ↓
before-images + Release Intent become durable
    ↓
adapter applies desired state
    ↓
adapter verifies target
    ↓
immutable Release created
    ↓
HEAD advances
    ↓
candidate validations join ledger suite
    ↓
released Changes can never enter another PR
    ↓
restart preserves all state
    ↓
historical Release can generate rollback PR
    ↓
rollback is reviewed and released forward as a new Release
```

That sequence gives the product one coherent correctness boundary at every stage: **durable Change acceptance, exclusive PR ownership, immutable review evidence, explicit approval, recoverable external application, immutable Release history, and forward-only rollback**. It keeps the architecture aligned with the original V3 goal—one local gyrif repository, one SQLite authority for governance state, immutable objects for durable history, adapter-specific correctness at the mutable database boundary, and a user experience centred on **Change → Context PR → Release** rather than exposing the storage machinery underneath. fileciteturn0file0