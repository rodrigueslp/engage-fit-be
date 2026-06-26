CREATE TABLE checkins (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    checkin_time TIME,
    source VARCHAR(32) NOT NULL,
    import_history_id UUID NOT NULL REFERENCES import_histories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT checkins_source_check CHECK (source IN ('wellhub', 'totalpass'))
);

CREATE INDEX idx_checkins_box_date ON checkins(box_id, checkin_date);
CREATE INDEX idx_checkins_box_student_date ON checkins(box_id, student_id, checkin_date);
CREATE INDEX idx_checkins_box_source_date ON checkins(box_id, source, checkin_date);
