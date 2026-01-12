-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfume_similarities (
    perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    similar_perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    similarity_score DECIMAL(5, 4) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (perfume_id, similar_perfume_id),
    CHECK (perfume_id != similar_perfume_id)
);

CREATE INDEX idx_perfume_similarities_perfume_id ON perfume_similarities(perfume_id, similarity_score DESC);
CREATE INDEX idx_perfume_similarities_similar_perfume_id ON perfume_similarities(similar_perfume_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfume_similarities;
-- +goose StatementEnd
