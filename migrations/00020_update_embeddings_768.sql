-- +goose Up
-- +goose StatementBegin

-- Update search embedding from 384 to 768 dimensions
-- First drop the index, then alter the column, then recreate the index
DROP INDEX IF EXISTS idx_perfume_search_embedding_hnsw;

ALTER TABLE perfume_search 
    ALTER COLUMN embedding TYPE VECTOR(768);

-- Add recommendation embedding column (768 dimensions)
ALTER TABLE perfume_search 
    ADD COLUMN IF NOT EXISTS rec_embedding VECTOR(768);

-- Recreate HNSW indexes for both embeddings
CREATE INDEX idx_perfume_search_embedding_hnsw 
    ON perfume_search USING hnsw(embedding vector_cosine_ops);

CREATE INDEX idx_perfume_search_rec_embedding_hnsw 
    ON perfume_search USING hnsw(rec_embedding vector_cosine_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_perfume_search_rec_embedding_hnsw;
DROP INDEX IF EXISTS idx_perfume_search_embedding_hnsw;

ALTER TABLE perfume_search DROP COLUMN IF EXISTS rec_embedding;

ALTER TABLE perfume_search ALTER COLUMN embedding TYPE VECTOR(384);

CREATE INDEX idx_perfume_search_embedding_hnsw 
    ON perfume_search USING hnsw(embedding vector_cosine_ops);

-- +goose StatementEnd
