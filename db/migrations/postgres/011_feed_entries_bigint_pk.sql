-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_entries ADD COLUMN seq bigint GENERATED ALWAYS AS IDENTITY;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE published       DROP CONSTRAINT published_item_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_embeddings DROP CONSTRAINT item_embeddings_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_chunks     DROP CONSTRAINT item_chunks_item_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    DROP CONSTRAINT feed_entries_pkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    ADD CONSTRAINT feed_entries_id_key UNIQUE (id);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    ADD PRIMARY KEY (seq);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE published       ADD CONSTRAINT published_item_id_fkey     FOREIGN KEY (item_id) REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_embeddings ADD CONSTRAINT item_embeddings_id_fkey    FOREIGN KEY (id)      REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_chunks     ADD CONSTRAINT item_chunks_item_id_fkey   FOREIGN KEY (item_id) REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE published       DROP CONSTRAINT published_item_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_embeddings DROP CONSTRAINT item_embeddings_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_chunks     DROP CONSTRAINT item_chunks_item_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    DROP CONSTRAINT feed_entries_pkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    DROP CONSTRAINT feed_entries_id_key;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    ADD PRIMARY KEY (id);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE feed_entries    DROP COLUMN seq;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE published       ADD CONSTRAINT published_item_id_fkey     FOREIGN KEY (item_id) REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_embeddings ADD CONSTRAINT item_embeddings_id_fkey    FOREIGN KEY (id)      REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE item_chunks     ADD CONSTRAINT item_chunks_item_id_fkey   FOREIGN KEY (item_id) REFERENCES feed_entries(id) ON DELETE CASCADE;
-- +goose StatementEnd
