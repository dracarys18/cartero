-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    hash TEXT NOT NULL,
    source TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS published (
    item_id TEXT NOT NULL,
    target TEXT NOT NULL,
    published_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (item_id, target),
    FOREIGN KEY (item_id) REFERENCES items(id)
);

CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash);
CREATE INDEX IF NOT EXISTS idx_items_source ON items(source);
CREATE INDEX IF NOT EXISTS idx_items_timestamp ON items(timestamp);
CREATE INDEX IF NOT EXISTS idx_published_item_id ON published(item_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_published_item_id;
DROP INDEX IF EXISTS idx_items_timestamp;
DROP INDEX IF EXISTS idx_items_source;
DROP INDEX IF EXISTS idx_items_hash;
DROP TABLE IF EXISTS published;
DROP TABLE IF EXISTS items;
-- +goose StatementEnd
