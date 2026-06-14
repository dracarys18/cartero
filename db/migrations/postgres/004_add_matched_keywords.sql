-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN IF NOT EXISTS matched_keywords TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS matched_keywords;
-- +goose StatementEnd
