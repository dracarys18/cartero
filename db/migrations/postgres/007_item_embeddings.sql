-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE item_embeddings (
    id         TEXT PRIMARY KEY REFERENCES feed_entries(id) ON DELETE CASCADE,
    embedding  halfvec(1024) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS item_embeddings;
-- +goose StatementEnd
