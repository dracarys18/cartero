-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries
    ADD COLUMN IF NOT EXISTS fts tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(title, '') || ' ' ||
            coalesce(description, '') || ' ' ||
            coalesce(content, ''))
    ) STORED;

CREATE INDEX IF NOT EXISTS feed_entries_fts_idx ON feed_entries USING GIN (fts);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS feed_entries_fts_idx;
ALTER TABLE feed_entries DROP COLUMN IF EXISTS fts;
-- +goose StatementEnd
