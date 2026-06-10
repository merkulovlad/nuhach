-- +goose Up
-- +goose StatementBegin

-- user_interactions is maintained as the latest value for each interaction type.
DELETE FROM user_interactions older
USING user_interactions newer
WHERE older.user_id = newer.user_id
  AND older.perfume_id = newer.perfume_id
  AND older.interaction_type = newer.interaction_type
  AND (older.created_at, older.id) < (newer.created_at, newer.id);

DROP INDEX IF EXISTS idx_user_interactions_composite;
CREATE UNIQUE INDEX idx_user_interactions_composite
    ON user_interactions(user_id, perfume_id, interaction_type);

-- User embeddings are derived data and can be regenerated from interactions.
TRUNCATE TABLE user_embeddings;
ALTER TABLE user_embeddings ALTER COLUMN embedding TYPE VECTOR(768);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

TRUNCATE TABLE user_embeddings;
ALTER TABLE user_embeddings ALTER COLUMN embedding TYPE VECTOR(384);

DROP INDEX IF EXISTS idx_user_interactions_composite;
CREATE INDEX idx_user_interactions_composite
    ON user_interactions(user_id, perfume_id, interaction_type);

-- +goose StatementEnd
