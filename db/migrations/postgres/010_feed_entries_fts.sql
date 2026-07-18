-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feed_entries_search_tsv
    ON feed_entries USING GIN (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_search_tsv;
-- +goose StatementEnd
