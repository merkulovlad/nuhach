-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_interactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    perfume_id BIGINT NOT NULL REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    interaction_type TEXT NOT NULL,
    interaction_score DECIMAL(3, 2) DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_interactions_user_id ON user_interactions(user_id, created_at DESC);
CREATE INDEX idx_user_interactions_perfume_id ON user_interactions(perfume_id);
CREATE INDEX idx_user_interactions_type ON user_interactions(interaction_type);
CREATE INDEX idx_user_interactions_composite ON user_interactions(user_id, perfume_id, interaction_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_interactions;
-- +goose StatementEnd
