-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_paginate;
CREATE INDEX IF NOT EXISTS idx_feed_entries_created_at
    ON feed_entries(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_created_at;
-- +goose StatementEnd
