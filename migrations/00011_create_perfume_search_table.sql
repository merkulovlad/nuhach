-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfume_search (
    id BIGSERIAL PRIMARY KEY,
    perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    doc_id TEXT NOT NULL,
    search_text TEXT NOT NULL,
    search_text_en TEXT NOT NULL,
    embedding VECTOR(384),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_perfume_search_perfume_id ON perfume_search(perfume_id);
CREATE UNIQUE INDEX idx_perfume_search_doc_id ON perfume_search(doc_id);
CREATE INDEX idx_perfume_search_text_ru_gin ON perfume_search USING GIN(to_tsvector('russian', search_text));
CREATE INDEX idx_perfume_search_text_en_gin ON perfume_search USING GIN(to_tsvector('english', search_text_en));
CREATE INDEX idx_perfume_search_embedding_hnsw ON perfume_search USING hnsw(embedding vector_cosine_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfume_search;
-- +goose StatementEnd
