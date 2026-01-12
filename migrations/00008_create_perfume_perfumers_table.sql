-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfume_perfumers (
    perfume_id BIGINT REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    perfumer_id BIGINT REFERENCES perfumers(id) ON DELETE CASCADE,
    PRIMARY KEY (perfume_id, perfumer_id)
);

CREATE INDEX idx_perfume_perfumers_perfume_id ON perfume_perfumers(perfume_id);
CREATE INDEX idx_perfume_perfumers_perfumer_id ON perfume_perfumers(perfumer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfume_perfumers;
-- +goose StatementEnd
