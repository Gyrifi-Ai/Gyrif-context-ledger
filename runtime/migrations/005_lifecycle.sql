ALTER TABLE changes ADD COLUMN withdrawn_at TEXT;
ALTER TABLE changes ADD COLUMN withdrawn_reason TEXT;
ALTER TABLE ledgers ADD COLUMN archived_at TEXT;

CREATE INDEX ledgers_active_list ON ledgers(archived_at, created_at DESC, id DESC);
