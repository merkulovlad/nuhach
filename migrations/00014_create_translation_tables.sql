-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS note_translations (
    id BIGSERIAL PRIMARY KEY,
    note_id BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    translation_ru TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(note_id)
);

CREATE INDEX idx_note_translations_note_id ON note_translations(note_id);

CREATE TABLE IF NOT EXISTS accord_translations (
    id BIGSERIAL PRIMARY KEY,
    accord_id BIGINT NOT NULL REFERENCES accords(id) ON DELETE CASCADE,
    translation_ru TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(accord_id)
);

CREATE INDEX idx_accord_translations_accord_id ON accord_translations(accord_id);

CREATE TABLE IF NOT EXISTS brand_translations (
    id BIGSERIAL PRIMARY KEY,
    brand_id BIGINT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    translation_ru TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(brand_id)
);

CREATE INDEX idx_brand_translations_brand_id ON brand_translations(brand_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS brand_translations;
DROP TABLE IF EXISTS accord_translations;
DROP TABLE IF EXISTS note_translations;
-- +goose StatementEnd
