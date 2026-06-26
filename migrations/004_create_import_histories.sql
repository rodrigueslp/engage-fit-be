CREATE TABLE import_histories (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    source VARCHAR(32) NOT NULL,
    total_records INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT import_histories_source_check CHECK (source IN ('wellhub', 'totalpass'))
);

CREATE INDEX idx_import_histories_box_id ON import_histories(box_id);
CREATE INDEX idx_import_histories_box_source ON import_histories(box_id, source);
