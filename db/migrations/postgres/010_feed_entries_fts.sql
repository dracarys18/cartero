-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN IF NOT EXISTS search_tsv tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feed_entries_search_tsv ON feed_entries USING GIN (search_tsv);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_search_tsv;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS search_tsv;
-- +goose StatementEnd
