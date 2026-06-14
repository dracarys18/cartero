-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN IF NOT EXISTS image_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN IF EXISTS image_url;
-- +goose StatementEnd
