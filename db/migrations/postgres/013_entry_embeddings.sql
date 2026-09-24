-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS entry_embeddings (
    id        TEXT PRIMARY KEY REFERENCES feed_entries(id) ON DELETE CASCADE,
    model     TEXT NOT NULL,
    embedding vector NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS entry_embeddings;
-- +goose StatementEnd
