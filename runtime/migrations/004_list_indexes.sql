CREATE INDEX ledgers_list ON ledgers(created_at DESC, id DESC);
CREATE INDEX changes_list ON changes(ledger_id, created_at DESC, id DESC);
CREATE INDEX changes_status_list ON changes(ledger_id, status, created_at DESC, id DESC);
CREATE INDEX proposals_list ON proposals(ledger_id, created_at DESC, id DESC);
CREATE INDEX proposals_status_list ON proposals(ledger_id, status, created_at DESC, id DESC);
CREATE INDEX releases_list ON releases(ledger_id, created_at DESC, id DESC);
