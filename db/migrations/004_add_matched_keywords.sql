-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN matched_keywords TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN matched_keywords;
-- +goose StatementEnd
