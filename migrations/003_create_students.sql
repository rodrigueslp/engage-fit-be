CREATE TABLE students (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    source VARCHAR(32) NOT NULL,
    external_id VARCHAR(255),
    risk_status VARCHAR(32) NOT NULL DEFAULT 'active',
    risk_last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT students_source_check CHECK (source IN ('wellhub', 'totalpass')),
    CONSTRAINT students_risk_status_check CHECK (risk_status IN ('active', 'observing', 'paused', 'not_interested'))
);

CREATE INDEX idx_students_box_id ON students(box_id);
CREATE INDEX idx_students_box_source_external_id ON students(box_id, source, external_id);
CREATE INDEX idx_students_box_name ON students(box_id, name);
CREATE INDEX idx_students_box_phone ON students(box_id, phone);
CREATE INDEX idx_students_box_risk_status ON students(box_id, risk_status);
