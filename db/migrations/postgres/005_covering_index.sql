-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_created_at;

-- Full covering index including all columns added by 003 and 004
CREATE INDEX IF NOT EXISTS idx_feed_entries_paginate
    ON feed_entries(created_at DESC)
    INCLUDE (id, title, link, description, content, author, source,
             image_url, matched_keywords, published_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_entries_paginate;
CREATE INDEX IF NOT EXISTS idx_feed_entries_created_at
    ON feed_entries(created_at DESC);
-- +goose StatementEnd
