-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN hash TEXT;
ALTER TABLE feed_entries ADD COLUMN entry_timestamp DATETIME;

INSERT INTO feed_entries (id, hash, source, entry_timestamp, created_at)
  SELECT i.id, i.hash, i.source, i.timestamp, i.created_at
  FROM items i WHERE true
  ON CONFLICT(id) DO UPDATE SET hash = excluded.hash, entry_timestamp = excluded.entry_timestamp;

DROP TABLE IF EXISTS published;
CREATE TABLE published (
    item_id TEXT NOT NULL,
    target TEXT NOT NULL,
    published_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (item_id, target),
    FOREIGN KEY (item_id) REFERENCES feed_entries(id)
);
CREATE INDEX IF NOT EXISTS idx_published_item_id ON published(item_id);

DROP TABLE IF EXISTS items;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    hash TEXT NOT NULL,
    source TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash);
CREATE INDEX IF NOT EXISTS idx_items_source ON items(source);
CREATE INDEX IF NOT EXISTS idx_items_timestamp ON items(timestamp);

INSERT INTO items (id, hash, source, timestamp, created_at)
  SELECT id, hash, source, entry_timestamp, created_at
  FROM feed_entries WHERE hash IS NOT NULL;

ALTER TABLE feed_entries DROP COLUMN hash;
ALTER TABLE feed_entries DROP COLUMN entry_timestamp;
-- +goose StatementEnd
