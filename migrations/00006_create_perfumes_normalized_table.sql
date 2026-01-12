-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfumes_normalized (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    perfume_name TEXT NOT NULL,
    brand_id BIGINT REFERENCES brands(id) ON DELETE SET NULL,
    gender TEXT,
    rating_value DECIMAL(3, 2),
    rating_count INTEGER,
    year INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_perfumes_normalized_url ON perfumes_normalized(url);
CREATE INDEX idx_perfumes_normalized_brand_id ON perfumes_normalized(brand_id);
CREATE INDEX idx_perfumes_normalized_gender ON perfumes_normalized(gender);
CREATE INDEX idx_perfumes_normalized_rating ON perfumes_normalized(rating_value DESC, rating_count DESC);
CREATE INDEX idx_perfumes_normalized_year ON perfumes_normalized(year);
CREATE INDEX idx_perfumes_normalized_perfume_name ON perfumes_normalized(perfume_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfumes_normalized;
-- +goose StatementEnd
