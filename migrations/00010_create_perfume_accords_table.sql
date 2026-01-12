-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfume_accords (
    perfume_id BIGINT REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    accord_id BIGINT REFERENCES accords(id) ON DELETE CASCADE,
    position SMALLINT NOT NULL,
    PRIMARY KEY (perfume_id, accord_id, position)
);

CREATE INDEX IF NOT EXISTS idx_perfume_accords_perfume_id ON perfume_accords(perfume_id);
CREATE INDEX IF NOT EXISTS idx_perfume_accords_accord_id ON perfume_accords(accord_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfume_accords;
-- +goose StatementEnd
