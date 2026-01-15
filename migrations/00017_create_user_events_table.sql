-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tg_id BIGINT NOT NULL,
    perfume_id BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    rating SMALLINT,
    request_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_events_user_id ON user_events(user_id);
CREATE INDEX IF NOT EXISTS idx_user_events_user_perfume ON user_events(user_id, perfume_id);
CREATE INDEX IF NOT EXISTS idx_user_events_request_id ON user_events(request_id);
CREATE INDEX IF NOT EXISTS idx_user_events_type ON user_events(event_type);
CREATE INDEX IF NOT EXISTS idx_user_events_created_at ON user_events(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_events;
-- +goose StatementEnd
