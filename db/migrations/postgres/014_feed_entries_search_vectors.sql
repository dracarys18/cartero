-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries
    ADD COLUMN IF NOT EXISTS keyword_vector tsvector
        GENERATED ALWAYS AS (to_tsvector('english', COALESCE(matched_keywords, ''))) STORED;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries
    ADD COLUMN IF NOT EXISTS title_vector tsvector
        GENERATED ALWAYS AS (to_tsvector('english', COALESCE(title, ''))) STORED;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries
    ADD COLUMN IF NOT EXISTS body_vector tsvector
        GENERATED ALWAYS AS (to_tsvector('english',
            COALESCE(description, '') || ' ' || LEFT(COALESCE(content, ''), 20000))) STORED;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS feed_entries_keyword_vector_idx ON feed_entries USING GIN (keyword_vector);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS feed_entries_title_vector_idx ON feed_entries USING GIN (title_vector);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS feed_entries_body_vector_idx ON feed_entries USING GIN (body_vector);
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_search_tsv;
-- +goose StatementEnd
-- +goose StatementBegin
ANALYZE feed_entries;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feed_entries_search_tsv
    ON feed_entries USING GIN (to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(description, '')));
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS feed_entries_keyword_vector_idx;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS feed_entries_title_vector_idx;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS feed_entries_body_vector_idx;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS keyword_vector;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS title_vector;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS body_vector;
-- +goose StatementEnd
