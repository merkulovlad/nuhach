-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS accords (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accords_name ON accords(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS accords;
-- +goose StatementEnd
