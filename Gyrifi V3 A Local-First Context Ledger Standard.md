# Gyrifi V3: A Local-First Context Ledger Standard

## Architectural verdict and reconstructed requirements

**Architectural verdict.** Gyrifi V3 should be designed as a **local-first governance engine for mutable AI context**, not as a database, not as a database proxy, not as a distributed transaction system, and not as a Git wrapper. Its job is narrower and more valuable:

> **Gyrifi governs proposed mutations to a context store, makes those mutations reviewable and testable before they become live, applies approved mutations safely, records exactly what happened, and makes the resulting history inspectable and reversible.**

The V2 product thesis was already directionally right: Gyrifi owns the governance history rather than a duplicate of the user's corpus, uses sparse changes instead of database copies, and provides release and rollback semantics around the real backing store. fileciteturn0file0 V3 should take those product properties and build a substantially simpler machine underneath them.

For the V3 system described here, I would deliberately optimise for **one process, one machine, one Docker image at most, one SQLite file, one Gyrifi repository, and initially one backing database per ledger**. There should be no Temporal, Kafka, Redis, service mesh, control-plane cluster, embedded Git repository, mandatory Postgres instance, or other independently operated system in the core product. SQLite already provides transactional durability, serialised writes and, in WAL mode, concurrent readers with snapshot isolation; that is almost exactly the concurrency model a single-process Gyrifi needs. citeturn3search0turn3search2

The most important conceptual simplification is this:

```text
USER CONCEPTS

Ledger
  │
  ├── Change
  │
  ├── Proposal
  │
  └── Release
```

That is enough.

A **Change** is one proposed change to one logical context unit.

A **Proposal** is a user-selected collection of Changes being reviewed and tested. The UI may call this a **Context PR**, because that is intuitive, but there does not need to be another internal "PR Revision" domain entity.

A **Release** is the immutable, commit-like record produced when an approved Proposal is successfully applied. Every ledger has one `HEAD` pointing at its latest Release.

Checks, reviews, approvals, rollback data, physical adapter operations and object packs are **properties or implementation mechanisms around those three concepts**, not more concepts the user must learn.

This gives Gyrifi Git-like ergonomics without pretending that a mutable database is a filesystem.

### Reconstructed requirements

The system I would actually build from your description is:

| Dimension | V3 requirement |
|---|---|
| Product | Governance/version-control layer for AI context stores |
| Deployment | Offline-first; native binary or single Docker container |
| Process model | Monolith |
| Backing store | One database per ledger in V3 |
| Ledger meaning | Logical context domain, not a physical storage engine |
| Consumers | Many agents may use the database governed by a ledger |
| Write path | Governed writes enter through Gyrifi's API |
| Draft behaviour | Proposed changes do not modify the backing DB |
| Review | Users can select arbitrary subsets of pending changes |
| Evaluation | Proposal may be tested against an effective proposed state |
| Release | Approved proposal is applied through a database-specific adapter |
| History | What changed, who changed it, why, when, checks and approvals |
| Rollback | Reconstruct older desired state and release it as a new Release |
| Data ownership | User's database remains owner of the corpus |
| Gyrifi ownership | Change history, governance metadata and rollback material |
| Persistence | Local SQLite + immutable content-addressed objects |
| Long-term storage | Compressed immutable packs, local or optionally object storage |
| External Git | Optional export/integration, not required for correctness |
| High availability | Explicitly out of scope for V3 |

There are several requirements that should **not** be silently invented because they materially affect the architecture:

**Atomicity is currently unspecified.** "All changes become visible at once" is achievable if the target database gives the adapter an appropriate transaction or atomic version-switch primitive. It is *not* universally achievable for arbitrary databases when agents query those databases directly. For example, Qdrant documents that point operations do not go through its Raft consensus layer, that update failures may be partially applied depending on consistency configuration, and that stronger write ordering is a separate configuration choice. citeturn7search0turn7search6

**The exact rollback promise is unspecified.** "Rollback any release from the last five years" and "the ledger must remain smaller than the database" cannot both be guaranteed for arbitrary workloads. That is a mathematical constraint rather than an implementation deficiency.

Suppose the database contains one 1 MB logical value, and that value is replaced every day with independent, incompressible content. After five years the live database still contains approximately 1 MB, while enough information for exact restoration of every historical version approaches 1,825 MB. No lossless storage format can guarantee compressing arbitrary independent information back below the current 1 MB database.

Therefore V3 needs an explicit product distinction:

```text
Audit retention      → cheap metadata + hashes retained indefinitely
Rollback retention   → actual before-value bytes retained for a configured window/budget
Full-history mode    → exact old values retained indefinitely, storage unbounded
```

That one decision will prevent years of confusion around compaction.

Other missing evidence should become benchmark inputs rather than architectural guesses: typical payload size, percentage of the database changed per month, number of changes per Proposal, number of ledgers per installation, expected sustained ingestion rate, maximum acceptable release time, first supported database adapter, whether direct writes outside Gyrifi are forbidden, required cryptographic signing model, and whether historical content itself carries compliance-retention obligations.

The architectural default should remain modest until those numbers demonstrate otherwise.

## System model, terminology and invariants

A Gyrifi installation should contain one **Gyrifi Repository**. The repository can contain multiple independent Ledgers.

```text
                         GYRIFI REPOSITORY

   ┌─────────────────────────────────────────────────────────┐
   │                                                         │
   │   Ledger: "Customer Support Context"                    │
   │   ├── Adapter: Qdrant support collection                │
   │   ├── HEAD: release r_9fd21...                          │
   │   ├── Pending Changes                                   │
   │   ├── Open Proposals                                    │
   │   └── Release History                                   │
   │                                                         │
   │   Ledger: "Product Knowledge"                           │
   │   ├── Adapter: another backing store                    │
   │   └── independent HEAD                                  │
   │                                                         │
   └─────────────────────────────────────────────────────────┘
                          │
                    adapters
                          │
             ┌────────────┴────────────┐
             ▼                         ▼
       User database             future database
```

The ledger is logical because the human-facing unit must survive changes in the physical representation underneath it.

For example:

```text
Logical Unit

policy/cancellation/enterprise
        │
        ├── Qdrant point 192
        ├── Qdrant point 193
        ├── Qdrant point 194
        └── metadata payload

User sees:
"Enterprise cancellation policy"

Gyrifi adapter sees:
three physical upserts and one deletion
```

That separation is fundamental.

### The three user-visible objects

**Change**

```json
{
  "change_id": "chg_01J...",
  "ledger": "support",
  "sequence": 8234,
  "unit": "policy/cancellation/enterprise",
  "action": "put",
  "value": "<object reference>",
  "base_fingerprint": "sha256:...",
  "author": "alice@example.com",
  "received_at": "...",
  "idempotency_key": "..."
}
```

A Change should describe the **desired logical value**, not low-level Qdrant/SQL/graph operations.

That prevents a user from accidentally selecting half of a semantic modification.

It also means `Change` is not gratuitous complexity. It replaces a much larger amount of hidden complexity. There has to be *some* object representing the thing the user is selecting when they say:

> "I received 100 updates; I want these 60."

Calling that object `Change` is probably the least surprising possible terminology.

**Proposal**

```json
{
  "proposal_id": "pr_142",
  "ledger": "support",
  "base": "r_9fd21...",
  "changes": [
    "chg_...",
    "chg_...",
    "chg_..."
  ]
}
```

A Proposal is mutable while the user is assembling it.

The implementation continuously computes:

```text
proposal_hash = SHA256(canonical Proposal contents)
```

There is therefore no need for a user-visible `PRRevision` object.

When someone edits the Proposal:

```text
old hash → no longer current
new hash → current proposal
```

Checks and approvals simply record:

```text
approved proposal_hash = 6ad8...
```

If the Proposal changes, those approvals no longer match.

This gets the safety property we wanted from "immutable PR revisions" without introducing another domain noun.

For the canonical representation, V3 should use an existing deterministic encoding rather than inventing one. RFC 8785 defines a JSON Canonicalization Scheme specifically so independently produced JSON can be reliably hashed and signed. citeturn9search0

**Release**

A Release is Gyrifi's equivalent of a Git commit.

```json
{
  "type": "release",
  "format": 1,
  "ledger": "support",
  "parent": "sha256:...",
  "proposal": "sha256:...",
  "changes": ["chg_...", "chg_..."],
  "before_images": "sha256:...",
  "adapter": {
    "kind": "qdrant",
    "protocol_version": 1
  },
  "released_by": "alice@example.com",
  "released_at": "...",
  "reason": "Updated cancellation policy"
}
```

Its identity is simply:

```text
release_id = SHA256(canonical Release object)
```

That is the simpler version of the "digest that connects everything."

There is no giant manually concatenated expression involving eleven hashes.

Instead:

> **Everything that must define the Release goes inside the canonical Release object; the hash of that object is its identity.**

This closely follows the useful part of Git's object model: Git gives immutable objects content-derived identities and then uses refs as mutable names pointing at those objects. Git stores loose objects initially and can later pack them without changing object identity. citeturn2search4turn0search0

### State ownership

Gyrifi should explicitly own three different kinds of state.

| State | Authority |
|---|---|
| Actual context corpus | User's database |
| Desired/governed history | Gyrifi Ledger |
| Current released Gyrifi version | Ledger `HEAD` |
| Pending changes and proposals | Gyrifi |
| User database's internal indexes | User database |
| Agent runtime state | Agent system, not Gyrifi |
| Historical before-values inside rollback window | Gyrifi object store |

This avoids turning agents into another state-management problem.

Your preferred relationship is the right one:

```text
                  Ledger
                    │
              governs one DB
                    │
                    ▼
               Context DB
              /     |      \
             /      |       \
        Agent A  Agent B   Agent C
```

V3 does **not** need to model:

```text
Agent A → Release 42
Agent B → Release 41
Agent C → Release 38
```

unless versioned reads later become an actual product requirement.

For V3, agents simply consume the current backing database. A release event can be exposed through SSE/webhooks so consumers can react to changes, but subscriber state is not part of the ledger.

### Critical invariants

The entire implementation should be reviewable against a small set of invariants:

1. An accepted API request is either durably represented as a Change or reported as failed.

2. Repeating an accepted request with the same idempotency key cannot create a second Change.

3. Every Change addresses a stable logical unit.

4. A Proposal contains explicit Change IDs; unrelated pending changes cannot accidentally enter it.

5. Checks and approvals are always associated with an exact Proposal hash.

6. A Release is immutable and has exactly one parent.

7. `HEAD` never names a Release that Gyrifi believes failed to apply.

8. Before modifying a target unit, Gyrifi knows the fingerprint of the value it expects to replace.

9. A mismatch between expected and actual target state is a conflict, never an unconditional overwrite.

10. Rollback never rewrites historical Releases.

11. Packing, compression, indexing and garbage collection are storage maintenance operations and never alter logical history.

12. Losing an acceleration index must not change semantic history.

Those are the rules around which code should be organised.

## How users use Gyrifi and what happens internally

The cleanest V3 experience should look almost boring.

### The user creates a ledger

Conceptually:

```bash
gyrifi ledger create support-context \
  --adapter qdrant \
  --collection support
```

The user supplies the database credentials separately through configuration or secrets.

Internally Gyrifi creates:

```text
Ledger
├── ID
├── name
├── adapter configuration
├── logical-unit schema
├── empty Change inbox
└── HEAD
```

No Git repository is created.

No Temporal workflow is created.

No database clone is created.

No per-agent state is created.

### Applications submit proposed writes

The application's write path becomes:

```text
Application
    │
    │ proposed change
    ▼
Gyrifi API
    │
    ├── validate
    ├── identify logical unit
    ├── deduplicate request
    ├── fingerprint current DB value
    ├── store proposed value
    └── record Change
```

For example:

```http
POST /v1/ledgers/support/changes

{
  "unit": "policy/refund/standard",
  "action": "put",
  "value": {
    "refund_days": 3
  },
  "idempotency_key": "refund-policy-2026-08-08"
}
```

Internally the backing adapter may need to read the current database state.

It does **not** write it.

The result might be:

```text
chg_18
base = sha256(value currently in DB)
after = sha256(proposed value)
status = pending
```

A request that retries because its HTTP connection died receives `chg_18` again rather than creating `chg_19`.

### The user sees an inbox

Suppose 100 Changes arrive.

```text
Pending Changes: 100

[x] Enterprise cancellation period     30d → 60d
[x] Refund period                       7d → 3d
[ ] Internal escalation threshold      3   → 4
[x] Warranty policy                     ...
...
```

This UI is served from indexed SQLite state.

It does not reconstruct the screen by scanning compressed journal files.

### The user selects 60 and opens a Proposal

```text
Proposal PR-142

Base: Release r_91fa...
Changes: 60
Affected logical units: 54
```

The other 40 remain pending.

No log segments are moved.

No segment is sealed.

No content is copied between "draft branches."

The Proposal is just a durable selection:

```text
Proposal → [Change 1, Change 7, Change 8, ...]
```

If several selected Changes target the same logical unit, Gyrifi should normalise them into one final desired state before evaluation.

For V3, I would make the internal Change model **state-based rather than command-based** whenever possible:

```text
prefer:
    "unit X should become value V"

over:
    "increment X"
    "append Y"
    "run arbitrary patch P"
```

State-based changes are naturally idempotent and considerably easier to compare, merge, retry and roll back.

### Gyrifi creates the effective proposed state

The CoW concept remains, but its definition becomes precise:

```text
Effective Proposal =
    current DB state
    with Proposal's logical Changes overlaid
```

Not:

```text
base + everything in an active OpLog segment
```

The adapter determines how to implement that overlay efficiently.

For a normal unit lookup:

```text
read(unit):
    if Proposal contains unit:
        return proposed value
    return adapter.read(unit)
```

For search/query evaluation, fidelity depends on the database, which matters enough to expose explicitly.

For Qdrant, the fast mechanism you described is genuinely useful:

```text
production search excluding changed IDs
        +
locally score changed/new vectors
        +
merge top-k
```

Qdrant supports `must_not` filters and ID filters, so exclusion itself maps naturally onto its API. citeturn4search2

But V3 should distinguish two modes.

**Fast Preview**

```text
ANN query excluding changed IDs
+
local scoring of proposed vectors
```

This is efficient but **not mathematically identical** to running the new collection.

Qdrant dense-vector search normally uses approximate HNSW unless exact search is requested; Qdrant's documentation explicitly notes that approximate rankings can change between requests and that filtering affects how the search graph is traversed. citeturn4search0turn4search4

**Reference Preview**

```text
exact database query excluding changed IDs
+
exact scoring of proposed vectors
+
merge
```

Qdrant's `exact=true` mode bypasses approximate HNSW and scans vectors to obtain exact results, at a potentially much higher cost. citeturn4search0turn4search8

That suggests the clean product contract:

```text
Preview fidelity
────────────────────────────────
FAST        production-like, approximate
REFERENCE   exact where adapter can provide it
UNSUPPORTED adapter cannot emulate safely
```

This is substantially better than claiming "Exclude-and-Inject exactly reproduces production" across every storage engine.

For a transactional relational database, the implementation can be different entirely: where safe, the adapter can create a transaction, apply candidate mutations, execute evaluation queries inside that transaction and roll it back. The important abstraction is therefore **"preview this Proposal"**, not **"Exclude-and-Inject"**.

### Review and approval

Every evaluation result contains:

```text
proposal_hash
check identity
check version
result
timestamp
```

Every approval contains:

```text
proposal_hash
actor
timestamp
```

If a Change is added or removed:

```text
Proposal hash changes
        ↓
old checks = stale
old approval = stale
```

No separate revision object is necessary.

### Release

The release algorithm should be intentionally conservative.

```text
1. Lock one ledger for release
2. Read current HEAD
3. Compare Proposal base with HEAD
4. Resolve/reject conflicts
5. Compile logical Changes into physical operations
6. Re-read affected target values
7. Verify expected fingerprints
8. Capture required before-images
9. Persist Release Intent
10. Apply target operations
11. Verify resulting target state
12. Create immutable Release object
13. Advance HEAD
14. Mark selected Changes released
15. Emit release event
```

The important part is steps 6–9.

A physical operation should look conceptually like:

```text
unit: point/42

expected:
    sha256:abc...

desired:
    sha256:def...

operation:
    upsert(...)
```

That makes release optimistic rather than blind.

### Failure walkthroughs

Consider a client that sends the same write five times because its network connection repeatedly dies.

SQLite stores the idempotency key under a unique constraint. The first transaction creates the Change; the next four resolve to the original Change. SQLite serialises writers and provides transaction isolation, so the monolithic engine does not need a second locking system to implement this invariant. citeturn3search0turn3search2

Consider Gyrifi crashing while recording a Change.

The proposed payload object is written first to a temporary file, fsynced, then renamed into the content-addressed object directory. Only afterwards does the SQLite transaction reference that hash.

```text
crash before rename:
    no object, no Change

crash after rename but before SQLite commit:
    unreachable object
    later garbage-collected

crash after SQLite commit:
    Change references an already durable object
```

This ordering deliberately prefers harmless garbage over dangling references.

Consider two users trying to release different Proposals from the same base.

Only one release operation should execute for a ledger at a time. The first advances HEAD. The second sees that:

```text
proposal.base != HEAD
```

and performs a logical rebase check.

For each changed unit:

```text
actual DB fingerprint == expected base
    → safe

actual DB fingerprint == desired value
    → already applied / no-op candidate

otherwise
    → conflict
```

No Git branch or distributed compare-and-swap is required in a single-process V3.

Consider Gyrifi crashing after the backing database commits but before SQLite records the new HEAD.

This is the most important crash window in the architecture because SQLite and the user's database do not participate in one common transaction.

Before applying anything, Gyrifi has already durably stored:

```text
Release Intent
├── proposal hash
├── expected fingerprints
├── desired fingerprints
├── before-image references
└── physical operation plan
```

On restart, Gyrifi sees an unfinished Release Intent and asks the adapter to classify the target:

```text
all target fingerprints = expected-before
    → application never took effect

all target fingerprints = desired-after
    → application succeeded; finish Release

mixed
    → resume safely or compensate; mark recovery state

unexpected third value
    → external drift/conflict; stop automatically
```

This is much more important than trying to manufacture "exactly once" execution across two independent storage systems.

Consider SQLite's WAL growing because a UI has left a read transaction open indefinitely.

SQLite's WAL documentation notes that long-lived readers can prevent checkpoint completion and allow the WAL to grow without bound. citeturn0search9 Gyrifi should therefore never hold UI read transactions across requests, should place explicit upper bounds on internal read lifetimes, and should expose WAL size/checkpoint metrics.

Consider a disk filling while Gyrifi is packing history.

The old loose objects are not deleted until the new pack has been completely written, fsynced, checksummed and registered.

A disk-full failure leaves:

```text
old representation still valid
partial temporary pack discarded
```

not a half-compacted repository.

That crash property should be designed before optimising compression.

## Refined architecture and why it snaps into place

The smallest coherent V3 architecture I would approve is this:

```text
                    ┌──────────────────────────────────┐
                    │          GYRIFI MONOLITH         │
                    │                                  │
 API / CLI / UI ───►│  Ingress                         │
                    │     │                            │
                    │     ▼                            │
                    │  Change Engine                   │
                    │     │                            │
                    │     ▼                            │
                    │  Proposal / Review Engine        │
                    │     │                            │
                    │     ├────► Evaluation Engine     │
                    │     │              │             │
                    │     │              ▼             │
                    │     │          DB Adapter        │
                    │     │                            │
                    │     ▼                            │
                    │  Release Engine ───────────────┐ │
                    │     │                         │ │
                    │     ▼                         ▼ │
                    │  SQLite                 DB Adapter│
                    │     │                         │  │
                    │     ▼                         │  │
                    │  Object Store                │  │
                    │     │                         │  │
                    │     ▼                         │  │
                    │  Background Packer            │  │
                    └─────┼─────────────────────────┼──┘
                          │                         │
                 local packs / optional S3         │
                                                    ▼
                                            USER DATABASE
```

There are only four meaningful internal subsystems:

```text
Repository
Adapter
Evaluation
Release
```

Everything else can remain packages/modules inside one executable.

### Repository

The on-disk repository should look approximately like:

```text
.gyrifi/
├── repository.json
├── state.db
├── objects/
│   ├── loose/
│   │   ├── ab/
│   │   │   └── cdef...
│   │   └── ...
│   └── packs/
│       ├── pack-a019....gpack
│       ├── pack-a019....idx
│       ├── pack-f812....gpack
│       └── pack-f812....idx
└── tmp/
```

`state.db` is SQLite.

It owns hot transactional metadata:

```text
ledgers
changes
proposals
proposal_changes
checks
approvals
release_intents
releases
ledger_heads
unit_heads
idempotency_keys
object_locations
pack_registry
events
```

SQLite is an unusually strong fit for V3 because it has serialisable transactions by serialising writers, supports multiple simultaneous readers, and WAL mode provides snapshot isolation while a writer is appending changes. citeturn3search0turn3search2

For a governance product I would configure WAL durability deliberately rather than accepting accidental defaults. SQLite documents that `synchronous=FULL` in WAL mode adds a sync after each transaction commit and provides durability across power loss, while `NORMAL` can lose the most recently committed transaction after power loss even though database consistency is preserved. citeturn3search1

For V3 the default should therefore favour:

```text
journal_mode = WAL
synchronous  = FULL
foreign_keys = ON
busy_timeout = bounded
```

unless benchmarks prove the fsync cost unacceptable for the actual workload.

This is governance metadata. Losing an acknowledged Change during power failure is usually worse than gaining a few milliseconds.

### Content-addressed object storage

Large values should not be stored repeatedly in SQLite rows.

Store immutable bytes under:

```text
SHA256(type || canonical metadata || content)
```

Object types might be only:

```text
VALUE
BEFORE_IMAGE
RELEASE
ATTESTATION
```

No more should be introduced until required.

The object store starts with **loose objects**.

Later a background packer moves them into compressed packs.

This deliberately adopts one of Git's best storage mechanisms without adopting Git's repository semantics. Git also begins with individually addressed objects and later consolidates them into compressed packfiles while retaining the same logical object identities. citeturn0search10turn0search0

A Gyrifi pack can be considerably simpler than a Git pack in its first version:

```text
GPack

Header
Object frame
Object frame
Object frame
...
Index footer
Pack checksum
```

Each entry contains:

```text
object hash
object type
uncompressed length
compressed length
compressed bytes
checksum
```

Zstandard is a sensible compression primitive because its format is stable and standardised, its decoder is designed for high throughput, and independent frames can be decompressed separately. Its seekable format demonstrates the useful pattern of separately compressed frames plus an index allowing subranges to be accessed without decompressing an entire archive. citeturn8search0turn8search2turn8search4

I would **not initially implement Git-style cross-object delta compression**.

Git's pack format supports objects encoded as deltas against other objects, and Git spends significant effort choosing useful delta bases and bounding associated costs. citeturn0search0turn0search5

Gyrifi should begin with:

```text
exact SHA-256 deduplication
+
zstd compression
+
immutable packs
```

Then measure.

Only if representative five-year workload simulations show storage remains unacceptable should V3.1 consider content-defined chunking or cross-object deltas.

This is an important application of the "smallest coherent system" principle: do not implement a miniature Git pack optimiser before evidence says you need one.

### The OpLog becomes the ledger journal, not another storage system

The useful OpLog idea should remain, but as a **logical journal inside the repository**, not a parallel authority.

Every durable lifecycle event gets a per-ledger monotonic sequence:

```text
8271 ChangeAccepted
8272 ChangeAccepted
8273 ProposalCreated
8274 ProposalChecked
8275 ProposalApproved
8276 ReleaseStarted
8277 ReleaseCompleted
```

The sequence number defines order.

The wall clock is metadata.

That protects the design from clock skew and timestamp collisions.

Recent journal rows live in SQLite.

Old journal ranges are archived into the same immutable pack mechanism as the rest of the repository.

Thus V3 has:

```text
ONE LEDGER HISTORY

hot:
    SQLite journal

cold:
    immutable packed journal records
```

not:

```text
OpLog history
+
Git history
+
SQLite history
```

### Why not use Git itself?

This is probably the most consequential design decision in V3.

**Use Git's proven mechanisms. Do not make Git the Gyrifi storage model.**

Git's native world is:

```text
blob
tree
commit
tag
ref
working tree
index
```

Objects represent complete filesystem content and trees; commits reference tree snapshots, and refs name commit positions. citeturn2search4

Gyrifi's native world is:

```text
external database
logical unit
proposed mutation
expected database fingerprint
before-image
adapter operation
evaluation result
release attempt
database application result
```

The fundamental difference is that Git possesses the content graph it versions.

Gyrifi intentionally does **not** possess the complete database it governs.

Trying to represent a 2 TB vector collection as a Git tree would violate the product's most important storage boundary.

Trying *not* to represent the database in Git means Git cannot itself compute the complete state, perform database-specific conflicts, produce before-images, or prove whether a target operation was applied.

You would therefore still need Gyrifi's own operational ledger beside Git.

That gives:

```text
Git commit graph
+
Gyrifi operation history
+
database state
```

and puts you straight back into multiple competing histories.

V3 should instead borrow exactly the mechanisms that fit:

| Git mechanism | Gyrifi equivalent |
|---|---|
| content-addressed object | Gyrifi object |
| commit | Release |
| parent commit | parent Release |
| HEAD/ref | ledger HEAD |
| working changes | Changes |
| index/staging | Proposal |
| packfile | GPack |
| `git log` | `gyrifi log` |
| `git show` | `gyrifi show` |
| `git diff` | `gyrifi diff` |
| `git fsck` | `gyrifi fsck` |
| `git gc` | `gyrifi gc` |
| `git revert` | `gyrifi rollback` |

Git demonstrates that refs can safely represent mutable names over immutable content, that objects can change physical representation through packing without changing identity, and that large repositories benefit from separate graph/index structures. Git's multi-pack index, for example, maps sorted object IDs to their pack and offset to keep lookups efficient as the number of packs grows; Git's changed-path Bloom filters accelerate history searches without changing canonical history. citeturn6search0turn6search2

Those are excellent mechanisms to study.

They do not require embedding Git itself.

An **optional Git exporter** can still be extremely valuable:

```text
gyrifi export git
```

could produce one signed JSON/Markdown artefact per Release in the user's normal engineering repository.

That gives GitHub/GitLab audit workflows without making Git part of Gyrifi's crash-consistency protocol.

### Highest-risk design commitments

The things V3 must get right from the beginning, in priority order, are:

| Priority | Commitment | Why difficult to change later |
|---|---|---|
| Critical | Stable logical-unit identity | Every diff, conflict and rollback depends on it |
| Critical | Honest release atomicity contract | Incorrect promise becomes API/product debt |
| Critical | Crash-recoverable DB apply protocol | Data corruption risk |
| Critical | Explicit rollback-retention contract | Determines long-term storage economics |
| High | Proposal hash / approval binding | Governance trust boundary |
| High | Adapter canonical fingerprinting | Determines conflict correctness |
| High | Immutable Release schema | Public history format |
| High | Pack format versioning | Five-year storage compatibility |
| Medium | Evaluation fidelity contract | Incorrect evaluations undermine release safety |
| Medium | Backpressure and storage quotas | Prevents the ledger becoming an unbounded queue |

The structural root behind all of these is the same:

> **Gyrifi is coordinating two fundamentally different worlds: immutable governance history and a mutable database it does not own.**

V3 becomes clean once every component acknowledges that boundary instead of attempting to hide it.

## Storage, compaction, rollback and evaluation

Long-term storage is where Gyrifi can become genuinely differentiated.

The design goal should not be "store every database state."

It should be:

> **Store exactly enough information to prove every governance decision and reverse recent governed mutations, while never duplicating unchanged database content.**

### What the ledger actually retains

For each released logical change:

```text
Unit ID
Before fingerprint
After fingerprint
Release ID
Author
Time
Reason
Check / approval metadata
Physical-operation summary
Before-value bytes if rollback-retained
```

The crucial trick is that Gyrifi generally does **not** need to retain every released after-value indefinitely.

Suppose:

```text
v0 → v1 → v2 → v3
```

The backing database currently contains `v3`.

For rollback history Gyrifi retains:

```text
Release 1 before-image: v0
Release 2 before-image: v1
Release 3 before-image: v2
```

To reconstruct `v1` from today's state:

```text
current DB = v3
apply reverse Release 3 → v2
apply reverse Release 2 → v1
```

That is the elegant part of chained before-images.

It lets Gyrifi avoid maintaining complete database snapshots.

But V3 should make one improvement:

> Before-image chains are the **storage mechanism**; rollback is expressed as a **new desired-state Release**.

### Rollback

Suppose:

```text
R40 → R41 → R42 → R43 → R44
                         HEAD = R44
```

The user requests:

```text
rollback to R41
```

Gyrifi walks backward:

```text
R44 before-images
R43 before-images
R42 before-images
```

and composes the desired values that existed at R41.

It then creates a normal Proposal:

```text
Proposal:
    restore logical state corresponding to R41
```

The standard evaluation/release machinery runs.

The result is:

```text
R40 → R41 → R42 → R43 → R44 → R45
                              ^
                              |
                        state restores R41
```

The Release object records:

```json
{
  "reverts_to": "R41"
}
```

This is much closer to `git revert` than `git reset`. Git explicitly distinguishes creating a new commit that undoes an earlier public commit from rewriting history by moving a branch pointer backwards. citeturn10search0turn10search1

Gyrifi should **never reset HEAD backwards** for released production history.

Rollback means forward history.

### Three different forms of compaction

The word *compaction* should have one meaning at a time.

V3 should distinguish:

```text
Proposal reduction
    Many selected Changes → final desired change per logical unit

Object packing
    Many loose immutable objects → compressed immutable GPack

History retention
    Old rollback payloads removed after policy/budget allows it
```

None of those equals Release.

Release is a governance/data transition.

Compaction is maintenance.

This distinction is important because both Kafka and etcd demonstrate why log retention/compaction exists: indefinitely retaining every superseded state creates unbounded storage growth. Kafka log compaction retains the latest value by key, while etcd explicitly compacts old revisions to prevent performance degradation and storage exhaustion. citeturn1search10turn1search4

Gyrifi has a different audit requirement, so it should **not delete Release history**.

Instead:

```text
Release metadata / hashes:
    permanent

Old rollback payload:
    retention governed

Physical archive representation:
    freely repackable
```

That separates auditability from unlimited historical-data retention.

### A concrete storage policy

I would ship three user-selectable policies:

| Policy | Audit metadata | Rollback payload | Intended use |
|---|---:|---:|---|
| `lean` | forever | last N days / storage budget | default |
| `extended` | forever | configured months/years | regulated or safety-sensitive |
| `complete` | forever | forever | user accepts unbounded growth |

Additionally:

```text
ledger.storage.max_bytes
ledger.storage.max_ratio_of_target
ledger.rollback.min_days
```

can define budgets.

When approaching the budget, Gyrifi should not silently delete recoverability.

It should report:

```text
Ledger storage: 83% of budget
Guaranteed exact rollback: 127 days
Projected exhaustion: 41 days
```

Then policy decides whether to:

```text
archive older packs
reduce rollback window
increase budget
stop accepting new Changes
```

This is a governance product; silent retention degradation would be unacceptable.

### Backpressure

All queues must be bounded.

Pending Changes occupy real disk.

Before-images occupy real disk.

Unreleased Proposals may pin otherwise collectible objects.

Therefore every ledger needs quotas such as:

```text
max_pending_changes
max_pending_bytes
max_proposal_bytes
max_object_bytes
max_rollback_bytes
```

When the configured bound is reached, Gyrifi rejects new proposed writes with an explicit resource-exhausted response rather than continuing until the machine runs out of disk.

SQLite itself can enforce a maximum page count and returns `SQLITE_FULL` when it cannot grow, but Gyrifi should apply its own domain-specific limits long before filesystem exhaustion. SQLite supports database sizes far beyond normal Gyrifi metadata requirements, so the meaningful limit should be operational policy rather than SQLite's theoretical ceiling. citeturn2search1

### Efficient history lookup

Do not use trigram indexing to answer:

```text
show history of exact logical unit X
```

SQLite should simply maintain:

```sql
CREATE INDEX history_by_unit
ON change_history(ledger_id, unit_key, sequence);
```

The permanent packed representation can contain a small per-pack unit index or Bloom filter.

Git uses changed-path Bloom filters specifically to avoid expensive history traversal for commits unlikely to touch a requested path. citeturn6search0

Gyrifi can apply the same underlying idea to logical-unit history.

### Evaluation needs an adapter fidelity contract

The adapter interface should be small:

```text
Adapter
├── read(unit)
├── fingerprint(unit)
├── compile(logical_changes)
├── preview(proposal, mode)
├── apply(release_intent)
├── verify(release_intent)
└── restore(plan)
```

And separately expose capabilities:

```text
supports_atomic_apply
supports_exact_preview
supports_conditional_write
supports_batch
supports_rollback
supports_version_switch
```

This prevents the abstraction from concealing database semantics.

For example, Qdrant has atomic alias switching between collections, which can be useful for blue/green collection strategies, but that requires a second collection and therefore solves a different problem from sparse in-place changes. citeturn4search7

For normal point mutation, Qdrant's own documentation says point updates do not use its Raft consensus system and may be partially applied under some failures. citeturn7search0turn7search6

Therefore the UI/API should honestly report an adapter's release guarantee:

```text
ATOMIC
    target database itself provides all-or-nothing application

RECOVERABLE
    target may expose intermediate state, but Gyrifi can safely
    resume or compensate after failure
```

Do not market both as the same thing.

If Gyrifi someday needs **universal atomic visibility** for arbitrary non-transactional stores, the architecture changes materially: agents would need to query through Gyrifi or honour a release/version indirection controlled by Gyrifi. That would turn Gyrifi partly into a data plane.

That is exactly the sort of complexity V3 should avoid until demanded.

## Proven mechanisms, trade-offs and scale path

V3 is intentionally not novel everywhere.

It should be novel in the **composition**.

### Mechanisms worth adopting

**SQLite WAL for hot control state.** SQLite's WAL mode allows readers and a writer to operate concurrently using snapshot isolation, while SQLite serialises writes rather than requiring application-level distributed locking. citeturn3search0turn0search9

**Content-addressed immutable objects from Git.** Object identity survives repacking, enables exact deduplication and makes corruption verifiable. Git uses hashes to name objects and pack files to make large object sets more space efficient. citeturn2search4turn0search10

**Parent-linked Release history from version control.** One immutable object references its predecessor; a mutable `HEAD` identifies current state. This makes history easy to inspect and reason about without storing a full materialised database snapshot for every release. Git's commit/ref model demonstrates the general mechanism. citeturn2search4turn2search0

**Background packing, not synchronous compaction.** Git's object storage demonstrates the useful property that new objects can remain loose and later be packed without changing their identities. Its multi-pack infrastructure exists precisely because large repositories cannot afford to rewrite one gigantic pack for every update. citeturn6search1turn6search5

**Bounded historical payload retention from MVCC/log systems.** etcd explicitly compacts obsolete historical revisions because retaining all versions indefinitely consumes resources; Kafka distinguishes ordinary retention from key-based log compaction. Gyrifi should similarly distinguish permanent audit records from configurable payload retention. citeturn1search4turn1search16

**Optimistic concurrency rather than global locking.** Each Change knows the target fingerprint against which it was prepared. Release verifies that fingerprint before applying. The principle resembles compare-and-swap: Git's `update-ref`, for example, can update a ref only when its existing object ID matches the expected old object ID. citeturn2search0

**Crash recovery through intent + reconciliation.** Because Gyrifi cannot atomically commit its SQLite transaction and an unrelated external database transaction, it records what it intends to do before touching the external store and makes application idempotent enough to determine after restart whether the desired state has been reached.

This last mechanism is the system's most important piece of distributed-systems reasoning, even though V3 itself is a monolith.

The network boundary to the user database makes the release protocol distributed whether or not the Gyrifi executable is distributed.

### Rejected alternatives

**Embedded Git as the primary ledger — reject.**

It solves immutable content and history well but has the wrong state model for database-specific mutation and would require a second operational record anyway.

Decision changes if Gyrifi eventually governs only file-backed context repositories and those files themselves are the source of truth.

**Custom database instead of SQLite — reject.**

You would be rebuilding journalling, transactions, indexes, locking, recovery, schema migration and corruption handling before Gyrifi's unique functionality exists.

SQLite already provides atomic transactions and crash-recovery mechanisms specifically engineered around local-file durability. citeturn3search3turn3search0

**Postgres in V3 Core — reject.**

Nothing in the stated V3 requirement requires multiple Gyrifi processes coordinating over a network.

Introduce a `MetadataStore` interface internally if it costs little, but ship SQLite only.

A Postgres implementation becomes rational when the requirement changes to:

```text
several Gyrifi processes
+
failover
+
remote shared metadata
+
multiple simultaneous writers on different hosts
```

Until then it increases operational state without increasing product capability.

**Temporal — reject for V3.**

A local persisted Release Intent plus a restartable state machine is enough.

Temporal becomes justified only when release execution must survive independent worker fleets, very long-running cross-service orchestration or distributed ownership.

**Kafka — reject.**

The per-ledger event journal is local state, not a high-fan-out distributed stream.

**Event sourcing as the whole application model — reject.**

Keep an event journal for audit and recovery, but keep normal SQLite tables for current state.

The UI should read:

```sql
SELECT * FROM changes WHERE status='pending'
```

not replay five years of events.

**Full manifests for every Release — reject.**

A ledger managing a 100 million-unit database should not write a 100 million-entry manifest because twelve units changed.

Store parent + patch and keep an indexed current fingerprint only for governed/touched units.

**Custom delta compression in the first release — reject.**

Exact CAS deduplication plus Zstandard packs is enough to establish the architecture. Git's sophisticated delta mechanisms are useful but come with non-trivial pack-planning and CPU complexity. citeturn0search5

### Scale path

The interesting scaling dimension is not "one billion users."

It is:

```text
number of accepted Changes
× changed payload bytes
× history duration
× churn of the same logical units
```

Gyrifi's path can therefore remain extremely boring for a long time.

**Initial scale**

```text
one process
one SQLite DB
local loose-object directory
one backing-store adapter
background thread for packing
```

**Growing history**

```text
SQLite keeps hot metadata and indexes
old object bodies move into immutable packs
old journal ranges move into packs
pack objects are zstd compressed
```

SQLite remains small because large payloads live outside it.

**Very large history**

```text
many immutable packs
SQLite object_location index
generation-based repacking
optional older-pack archive to S3-compatible storage
small local cache of recent packs
```

Git's multi-pack index is evidence that large immutable-object repositories benefit from indexing across packs rather than repeatedly repacking the entire object population. Git's MIDX offers logarithmic object lookup across many packs and now supports incremental layering. citeturn6search2turn6search1

Gyrifi does not need to implement MIDX itself; SQLite can perform the cross-pack lookup.

That is another place where using a boring embedded database eliminates custom infrastructure.

**High ingestion rate**

SQLite remains viable while the workload is fundamentally one local writer. If benchmarks eventually show individual fsyncs dominate, the first response should be bounded group commit/batching rather than replacing the architecture.

**Multiple local ledgers**

Add fair per-ledger scheduling and independent byte quotas so one highly active ledger cannot consume every release worker or every byte of rollback space.

**Remote archive**

Introduce:

```text
PackStore
├── LocalFilesystem
└── S3Compatible
```

Packs are uploaded only after they are immutable.

Do not maintain an appendable "active S3 segment."

Local storage remains the active persistence layer; remote object storage is archival.

**Multiple Gyrifi servers**

Only here does V4-style architecture become justified:

```text
SQLite → transactional network metadata store
local release lock → distributed lease/serialisation
local object packs → shared object store
```

The public Change/Proposal/Release model should survive unchanged.

That is how V3 avoids painting itself into a corner without prematurely building V4.

## Migration sequence and implementation plan

The migration should be organised around **establishing the new invariants**, not attempting to preserve every old internal abstraction.

**First, freeze the vocabulary and public model.**

Approve exactly:

```text
Ledger
Change
Proposal
Release
HEAD
```

Anything else needs a strong reason to become user-facing.

Define the canonical JSON schemas and hashing rules before implementation. Release format objects should carry an explicit format version from day one.

Validation gate:

```text
Two independent implementations serialising the same Release
must generate the same object hash.
```

RFC 8785 exists specifically to make deterministic JSON suitable for hashing and signing. citeturn9search0

**Next, build the repository.**

Implement:

```text
state.db
content-addressed loose objects
SQLite object index
atomic object writer
fsck
basic GC
```

No compression yet.

Validation gate:

```text
random process termination after every filesystem/SQLite step
must never produce a committed reference to a missing object
```

**Then implement Change ingestion.**

Requirements:

```text
stable logical-unit key
base fingerprint
desired-state representation
idempotency key
author
sequence
```

Do not build Proposal/evaluation before this representation feels unquestionably correct.

**Then implement Proposal.**

Proposal is only:

```text
metadata
+
selected Change IDs
+
base HEAD
+
hash
```

Implement:

```text
status
diff
add/remove Change
proposal hash
approval invalidation
```

**Then implement one adapter end to end.**

Do not implement Qdrant + Postgres + Memgraph + S3 simultaneously.

Choose the database most representative of the real product.

Implement:

```text
read
fingerprint
compile
preview
apply
verify
restore
```

and define its exact atomicity/recovery semantics.

**Then implement the Release Intent state machine.**

Persist every externally observable step.

For example:

```text
PREPARING
    ↓
READY
    ↓
APPLYING
    ↓
VERIFYING
    ↓
RELEASED
```

with:

```text
RECOVERY_REQUIRED
```

for states that cannot safely self-resolve.

Do not hide this state machine behind background jobs; make it directly inspectable with:

```bash
gyrifi release inspect <id>
```

**Then implement rollback.**

Only after normal release has excellent crash behaviour.

Rollback should reuse the normal Proposal and Release code path rather than gaining a separate mutation engine.

**Then implement evaluation.**

Start with an honest adapter fidelity:

```text
FAST
REFERENCE
UNSUPPORTED
```

For vector adapters, compare Sparse Query Interception results against a physically materialised reference environment before deciding which tests can safely gate production.

**Then build GPack.**

Start:

```text
loose object
      ↓
zstd frame
      ↓
immutable pack
      ↓
self index
      ↓
SQLite location index
```

Add pack verification before garbage collection.

Only then add:

```text
background repacking
pack generations
S3 archive
```

**Finally add optional Git export.**

By this point the Release object is already canonical.

Git export becomes trivial:

```text
Release JSON
checks
approvals
human summary
```

That is precisely where Git belongs: as a governance integration boundary rather than an internal dependency.

## Proof obligations before V3 can be trusted

The standard for Gyrifi should be stronger than "unit tests pass."

The correctness properties are sufficiently crisp that they should be actively attacked.

**Crash consistency.** Build failpoints after every durable operation in Change acceptance and Release. Kill the process at each point thousands of times.

The invariant after restart must always be one of:

```text
change absent
or
change completely present
```

and:

```text
release target unchanged
or
release target fully desired
or
release explicitly in recoverable/recovery-required state
```

Never:

```text
HEAD says released while target is known incomplete
```

SQLite itself is designed to preserve atomic transaction behaviour across crashes and power loss, but Gyrifi must separately test the protocol surrounding its filesystem objects and external database. citeturn3search3turn3search1

**Idempotency.** Generate duplicated, reordered and delayed client requests.

For N repetitions of one idempotency key:

```text
number of logical Changes = 1
```

For an interrupted Release resumed N times:

```text
resulting target state = one application of desired state
```

**Concurrency.** Release two Proposals prepared from the same HEAD simultaneously.

Exactly one may proceed from that base without a rebase decision.

No unrelated Change may be accidentally incorporated.

**Conflict correctness.** Mutate one affected database unit externally between Proposal creation and Release.

Gyrifi must detect:

```text
actual fingerprint != expected fingerprint
```

and refuse an unsafe overwrite.

**Rollback.** Generate long random histories with updates, inserts and deletes to the same logical units.

For every Release inside the retained rollback window:

```text
rollback(current, target)
```

must produce state equal to the historical target for every governed unit.

Then perform the same experiment with a conflicting external mutation and verify Gyrifi stops rather than destroys it.

**Pack correctness.** Randomly choose loose objects, pack them, delete the loose representation, and verify byte-for-byte identity after retrieval.

Flip random bits in:

```text
object
pack
pack index
SQLite location metadata
```

and require `gyrifi fsck` either to reconstruct the index or report the corrupted object precisely.

**Compaction crash tests.**

Kill Gyrifi during:

```text
pack construction
pack fsync
pack registration
loose-object deletion
pack merge
```

Every restart must retain at least one valid copy of every reachable object.

This is the same high-level property that makes Git's distinction between object identity and packed representation so valuable: packing may alter physical representation without changing the referenced object. citeturn0search0turn0search10

**SQLite WAL behaviour.** Hold deliberately slow readers while writing heavily and verify that:

```text
WAL bytes
checkpoint age
oldest read transaction
```

are visible and bounded by Gyrifi's operational logic. SQLite explicitly warns that continuously overlapping readers can prevent complete checkpoints and let the WAL grow indefinitely. citeturn0search9

**Disk exhaustion.** Inject filesystem-full errors at every write path.

An installation approaching its configured storage budget must first enter a visible pressure state, then reject new Changes before repository integrity is endangered.

**Five-year storage simulation.** This should be a release blocker.

Take representative production distributions for:

```text
logical-unit size
updates/day
same-unit churn
percentage deletes
proposal size
rollback retention
compressibility
```

and replay five simulated years.

Measure:

```text
raw changed bytes
unique CAS bytes
compressed pack bytes
SQLite bytes
index bytes
rollback bytes
audit-only bytes
```

Do not make a "ledger will be less than 10% of your DB" claim until this benchmark establishes it for a clearly defined workload.

The product should instead expose the honest equation:

```text
Ledger size
≈ pending proposed bytes
+ retained unique before-image bytes
+ permanent governance metadata
+ indexes
+ compression overhead
```

**Evaluation fidelity.** For every supported adapter, construct the same Proposal in two ways:

```text
A. Gyrifi Preview
B. actually materialised proposed database state
```

Run thousands of representative queries and compare outputs.

For Qdrant in particular, compare:

```text
Fast Overlay vs actual ANN state
Reference Overlay vs exact materialised state
```

because Qdrant explicitly distinguishes approximate HNSW search from exact full-scan search. citeturn4search0turn4search8

Only checks demonstrated to have acceptable fidelity should be allowed to act as mandatory release gates.

**Performance.** Measure rather than pre-declaring arbitrary RPS requirements.

At minimum benchmark independently:

```text
single Change acceptance
100-Change batch
10,000 pending Changes
Proposal with 1 / 100 / 10,000 Changes
unit-history lookup after millions of Changes
cold object retrieval from hundreds of packs
release preparation
before-image capture
rollback planning
repository startup
fsck
GC/repack
```

The important scaling requirement is that routine operations depend on the **size of the requested change**, not the total size of the user's database or the total age of the ledger.

That should become a formal V3 design objective:

```text
Submitting k changed logical units:
    O(k), excluding target-specific lookups

Creating a Proposal from k selected Changes:
    O(k log k) or better

Normal diff:
    proportional to changed units

Release preparation:
    proportional to affected physical units

Rollback over r releases:
    proportional to units changed across those releases

Reading current database:
    never requires replaying historical ledger

Packing:
    background and bounded by selected pack generation,
    never by mandatory full-history rewrite
```

And finally, one architectural acceptance test should sit above all the others:

> **An engineer who understands SQLite transactions, the Change/Proposal/Release model and one database Adapter should be able to debug almost any V3 failure without understanding the entire codebase.**

That is the strongest signal that the boundaries are correct.

The resulting system has a remarkably compact mental model:

```text
Writes arrive
    ↓
Gyrifi records Changes
    ↓
User selects Changes into a Proposal
    ↓
Proposal hash binds evaluation + approval
    ↓
Gyrifi verifies target fingerprints
    ↓
Before-images make the operation reversible
    ↓
Adapter applies the approved desired state
    ↓
Immutable Release becomes HEAD
    ↓
Loose historical objects are eventually packed
    ↓
Old rollback payload ages according to explicit policy
    ↓
Audit history remains
```

That is the V3 I would build.

Not Git sitting beside an OpLog beside a workflow engine beside a database.

A **single Gyrifi ledger repository**, using SQLite for hot transactional state, content-addressed immutable objects for durable history, compressed packs for age, stable logical units for human interaction, adapter-specific mechanisms for database correctness, and three user concepts—**Change, Proposal, Release**—that map directly onto what the system actually does.