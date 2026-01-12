-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS perfume_top_notes (
    perfume_id BIGINT REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    note_id BIGINT REFERENCES notes(id) ON DELETE CASCADE,
    PRIMARY KEY (perfume_id, note_id)
);

CREATE INDEX idx_perfume_top_notes_perfume_id ON perfume_top_notes(perfume_id);
CREATE INDEX idx_perfume_top_notes_note_id ON perfume_top_notes(note_id);

CREATE TABLE IF NOT EXISTS perfume_middle_notes (
    perfume_id BIGINT REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    note_id BIGINT REFERENCES notes(id) ON DELETE CASCADE,
    PRIMARY KEY (perfume_id, note_id)
);

CREATE INDEX idx_perfume_middle_notes_perfume_id ON perfume_middle_notes(perfume_id);
CREATE INDEX idx_perfume_middle_notes_note_id ON perfume_middle_notes(note_id);

CREATE TABLE IF NOT EXISTS perfume_base_notes (
    perfume_id BIGINT REFERENCES perfumes_normalized(id) ON DELETE CASCADE,
    note_id BIGINT REFERENCES notes(id) ON DELETE CASCADE,
    PRIMARY KEY (perfume_id, note_id)
);

CREATE INDEX idx_perfume_base_notes_perfume_id ON perfume_base_notes(perfume_id);
CREATE INDEX idx_perfume_base_notes_note_id ON perfume_base_notes(note_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS perfume_base_notes;
DROP TABLE IF EXISTS perfume_middle_notes;
DROP TABLE IF EXISTS perfume_top_notes;
-- +goose StatementEnd
