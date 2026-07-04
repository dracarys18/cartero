-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS item_embeddings_embedding_hnsw
    ON item_embeddings USING hnsw (embedding halfvec_cosine_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS item_embeddings_embedding_hnsw;
-- +goose StatementEnd
