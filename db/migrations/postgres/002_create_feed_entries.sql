-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS feed_entries (
    id               TEXT PRIMARY KEY,
    title            TEXT NOT NULL,
    link             TEXT,
    description      TEXT,
    content          TEXT,
    author           TEXT,
    source           TEXT NOT NULL,
    published_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- For ListRecentEntries ordered by published_at
CREATE INDEX IF NOT EXISTS idx_feed_entries_published_at
    ON feed_entries(published_at DESC);

CREATE INDEX IF NOT EXISTS idx_feed_entries_source
    ON feed_entries(source);

-- Basic index for pagination (covering index added in 005 after all columns exist)
CREATE INDEX IF NOT EXISTS idx_feed_entries_created_at
    ON feed_entries(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_created_at;
DROP INDEX IF EXISTS idx_feed_entries_source;
DROP INDEX IF EXISTS idx_feed_entries_published_at;
DROP TABLE IF EXISTS feed_entries;
-- +goose StatementEnd
