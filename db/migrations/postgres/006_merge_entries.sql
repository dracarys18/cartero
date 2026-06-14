-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN IF NOT EXISTS hash TEXT;
ALTER TABLE feed_entries ADD COLUMN IF NOT EXISTS entry_timestamp TIMESTAMPTZ;

INSERT INTO feed_entries (id, hash, source, entry_timestamp, created_at, title)
  SELECT i.id, i.hash, i.source, i.timestamp, i.created_at, ''
  FROM items i
  ON CONFLICT(id) DO UPDATE SET hash = EXCLUDED.hash, entry_timestamp = EXCLUDED.entry_timestamp;

ALTER TABLE published DROP CONSTRAINT IF EXISTS published_item_id_fkey;
ALTER TABLE published ADD CONSTRAINT published_item_id_fkey
  FOREIGN KEY (item_id) REFERENCES feed_entries(id) ON DELETE CASCADE;

DROP TABLE IF EXISTS items;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS items (
    id         TEXT PRIMARY KEY,
    hash       TEXT NOT NULL,
    source     TEXT NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash);
CREATE INDEX IF NOT EXISTS idx_items_source ON items(source);
CREATE INDEX IF NOT EXISTS idx_items_timestamp ON items(timestamp);

INSERT INTO items (id, hash, source, timestamp, created_at)
  SELECT id, hash, source, entry_timestamp, created_at
  FROM feed_entries WHERE hash IS NOT NULL;

ALTER TABLE published DROP CONSTRAINT IF EXISTS published_item_id_fkey;
ALTER TABLE published ADD CONSTRAINT published_item_id_fkey
  FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE;

ALTER TABLE feed_entries DROP COLUMN IF EXISTS hash;
ALTER TABLE feed_entries DROP COLUMN IF EXISTS entry_timestamp;
-- +goose StatementEnd
