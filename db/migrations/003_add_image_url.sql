-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN image_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feed_entries DROP COLUMN image_url;
-- +goose StatementEnd
