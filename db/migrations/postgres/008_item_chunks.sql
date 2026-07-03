-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE item_chunks (
    item_id     TEXT NOT NULL REFERENCES feed_entries(id) ON DELETE CASCADE,
    chunk_index INT  NOT NULL,
    embedding   halfvec(1024) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (item_id, chunk_index)
);

CREATE INDEX item_chunks_embedding_hnsw
    ON item_chunks USING hnsw (embedding halfvec_cosine_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS item_chunks;
-- +goose StatementEnd
