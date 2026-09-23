-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS item_chunks;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS item_embeddings;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS item_embeddings (
    id         TEXT PRIMARY KEY REFERENCES feed_entries(id) ON DELETE CASCADE,
    embedding  halfvec(1024) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS item_embeddings_embedding_hnsw
    ON item_embeddings USING hnsw (embedding halfvec_cosine_ops);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS item_chunks (
    item_id     TEXT NOT NULL REFERENCES feed_entries(id) ON DELETE CASCADE,
    chunk_index INT  NOT NULL,
    embedding   halfvec(1024) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (item_id, chunk_index)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS item_chunks_embedding_hnsw
    ON item_chunks USING hnsw (embedding halfvec_cosine_ops);
-- +goose StatementEnd
