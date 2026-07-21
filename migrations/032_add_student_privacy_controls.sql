ALTER TABLE students
    ADD COLUMN contact_status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    ADD COLUMN contact_status_updated_at TIMESTAMPTZ,
    ADD COLUMN contact_status_source VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN anonymized_at TIMESTAMPTZ,
    ADD CONSTRAINT students_contact_status_check CHECK (contact_status IN ('unknown', 'opted_in', 'opted_out'));

CREATE INDEX idx_students_box_contact_status ON students(box_id, contact_status);

CREATE TABLE privacy_suppressions (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    source VARCHAR(32) NOT NULL,
    external_id_hash CHAR(64) NOT NULL,
    reason VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT privacy_suppressions_source_check CHECK (source IN ('wellhub', 'totalpass')),
    CONSTRAINT privacy_suppressions_box_identity_unique UNIQUE (box_id, source, external_id_hash)
);

CREATE TABLE privacy_audit_events (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(40) NOT NULL,
    reason VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT privacy_audit_events_action_check CHECK (action IN ('exported', 'contact_preference_updated', 'anonymized'))
);

CREATE INDEX idx_privacy_audit_events_box_created ON privacy_audit_events(box_id, created_at DESC);
