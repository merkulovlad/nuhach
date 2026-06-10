-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS offer_search_jobs (
    id BIGSERIAL PRIMARY KEY,
    perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_offer_search_jobs_active_perfume
    ON offer_search_jobs(perfume_id)
    WHERE status IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS idx_offer_search_jobs_queue
    ON offer_search_jobs(status, requested_at);

CREATE TABLE IF NOT EXISTS store_offers (
    id BIGSERIAL PRIMARY KEY,
    perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    search_job_id BIGINT REFERENCES offer_search_jobs(id) ON DELETE SET NULL,
    store TEXT NOT NULL,
    seller TEXT,
    title TEXT NOT NULL,
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    old_price NUMERIC(12, 2),
    currency TEXT NOT NULL DEFAULT 'RUB',
    volume_ml INTEGER,
    concentration TEXT,
    product_type TEXT,
    in_stock BOOLEAN NOT NULL DEFAULT TRUE,
    url TEXT NOT NULL,
    rating NUMERIC(3, 2),
    reviews_count INTEGER,
    match_confidence NUMERIC(4, 3) NOT NULL DEFAULT 0,
    risk_level TEXT NOT NULL DEFAULT 'unknown',
    risk_score NUMERIC(4, 3) NOT NULL DEFAULT 0,
    comment TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (perfume_id, store, url)
);

CREATE INDEX IF NOT EXISTS idx_store_offers_perfume_expires
    ON store_offers(perfume_id, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_store_offers_price
    ON store_offers(perfume_id, price)
    WHERE in_stock = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS store_offers;
DROP TABLE IF EXISTS offer_search_jobs;
-- +goose StatementEnd
