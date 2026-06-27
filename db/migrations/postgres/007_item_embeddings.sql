-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE item_embeddings (
    id         TEXT PRIMARY KEY REFERENCES feed_entries(id) ON DELETE CASCADE,
    embedding  halfvec NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_item_embeddings_embedding
    ON item_embeddings USING ivfflat (embedding halfvec_cosine_ops) WITH (lists = 100);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_item_embeddings_embedding;
DROP TABLE IF EXISTS item_embeddings;
-- +goose StatementEnd
