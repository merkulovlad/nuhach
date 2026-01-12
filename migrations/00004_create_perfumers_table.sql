-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfumers (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_perfumers_name ON perfumers(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfumers;
-- +goose StatementEnd
