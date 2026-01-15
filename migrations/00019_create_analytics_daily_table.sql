-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS analytics_daily (
    id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    surface TEXT NOT NULL, -- 'search', 'recommendations'
    ctr DOUBLE PRECISION NOT NULL DEFAULT 0,
    precision_k DOUBLE PRECISION NOT NULL DEFAULT 0,
    recall_k DOUBLE PRECISION NOT NULL DEFAULT 0,
    map_k DOUBLE PRECISION NOT NULL DEFAULT 0,
    ndcg_k DOUBLE PRECISION NOT NULL DEFAULT 0,
    coverage DOUBLE PRECISION NOT NULL DEFAULT 0,
    diversity DOUBLE PRECISION NOT NULL DEFAULT 0,
    novelty DOUBLE PRECISION NOT NULL DEFAULT 0,
    impressions BIGINT NOT NULL DEFAULT 0,
    clicks BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, surface)
);

CREATE INDEX IF NOT EXISTS idx_analytics_daily_date ON analytics_daily(date);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_surface ON analytics_daily(surface);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS analytics_daily;
-- +goose StatementEnd
