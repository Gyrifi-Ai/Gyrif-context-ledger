ALTER TABLE changes ADD COLUMN prepare_owner TEXT;
ALTER TABLE changes ADD COLUMN prepare_claimed_at TEXT;
ALTER TABLE changes ADD COLUMN prepare_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE changes ADD COLUMN prepare_after TEXT;
ALTER TABLE changes ADD COLUMN invalid_reason TEXT;
ALTER TABLE changes ADD COLUMN noop INTEGER NOT NULL DEFAULT 0;

-- These states were defined but not produced before this migration; normalize any legacy rows.
UPDATE changes SET status = 'READY' WHERE status IN ('ACCEPTED', 'INVALID');
CREATE INDEX changes_preparation_queue ON changes(status, prepare_after, prepare_claimed_at, created_at, id);
