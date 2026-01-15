-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS impression_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    surface TEXT NOT NULL, -- 'search', 'recommendations', 'similar'
    perfume_ids BIGINT[] NOT NULL,
    ranks INTEGER[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_impression_logs_user_id ON impression_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_impression_logs_request_id ON impression_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_impression_logs_surface ON impression_logs(surface);
CREATE INDEX IF NOT EXISTS idx_impression_logs_created_at ON impression_logs(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS impression_logs;
-- +goose StatementEnd
