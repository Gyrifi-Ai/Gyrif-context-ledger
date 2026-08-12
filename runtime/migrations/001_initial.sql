CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledgers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_heads (
    ledger_id TEXT PRIMARY KEY REFERENCES ledgers(id) ON DELETE CASCADE,
    release_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS changes (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    unit_key TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('PUT', 'DELETE')),
    desired BLOB,
    base_fingerprint TEXT NOT NULL DEFAULT '',
    desired_fingerprint TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (ledger_id, sequence),
    UNIQUE (ledger_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS changes_inbox ON changes(ledger_id, status, sequence DESC);
CREATE INDEX IF NOT EXISTS changes_unit_history ON changes(ledger_id, unit_key, sequence DESC);

CREATE TABLE IF NOT EXISTS proposals (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    base_release_id TEXT NOT NULL DEFAULT '',
    proposal_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS proposals_by_ledger ON proposals(ledger_id, created_at DESC);

CREATE TABLE IF NOT EXISTS proposal_changes (
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    change_id TEXT NOT NULL REFERENCES changes(id),
    ordinal INTEGER NOT NULL,
    PRIMARY KEY (proposal_id, change_id),
    UNIQUE (change_id)
);

CREATE TABLE IF NOT EXISTS checks (
    id TEXT PRIMARY KEY,
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    proposal_hash TEXT NOT NULL,
    kind TEXT NOT NULL,
    passed INTEGER NOT NULL,
    summary TEXT NOT NULL,
    evidence BLOB,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS checks_current ON checks(proposal_id, proposal_hash, created_at DESC);

CREATE TABLE IF NOT EXISTS approvals (
    id TEXT PRIMARY KEY,
    proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    proposal_hash TEXT NOT NULL,
    actor TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (proposal_id, proposal_hash, actor)
);

CREATE TABLE IF NOT EXISTS release_intents (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id),
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    proposal_hash TEXT NOT NULL,
    parent_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    plan BLOB NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS unfinished_intents ON release_intents(status, created_at);

CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL REFERENCES ledgers(id),
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    parent_id TEXT NOT NULL DEFAULT '',
    release_hash TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS releases_by_ledger ON releases(ledger_id, created_at DESC);
