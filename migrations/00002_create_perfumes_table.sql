-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfumes (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    perfume_name TEXT NOT NULL,
    brand TEXT NOT NULL,
    country TEXT,
    gender TEXT,
    rating_value DECIMAL(3, 2),
    rating_count INTEGER,
    year INTEGER,
    top_notes TEXT,
    middle_notes TEXT,
    base_notes TEXT,
    perfumer1 TEXT,
    perfumer2 TEXT,
    main_accord1 TEXT,
    main_accord2 TEXT,
    main_accord3 TEXT,
    main_accord4 TEXT,
    main_accord5 TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_perfumes_url ON perfumes(url);
CREATE INDEX idx_perfumes_brand ON perfumes(brand);
CREATE INDEX idx_perfumes_gender ON perfumes(gender);
CREATE INDEX idx_perfumes_rating ON perfumes(rating_value DESC, rating_count DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfumes;
-- +goose StatementEnd
