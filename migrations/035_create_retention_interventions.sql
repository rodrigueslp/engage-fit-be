CREATE TABLE retention_interventions (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    channel VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'planned',
    outcome VARCHAR(32),
    planned_for TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    notes VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT retention_interventions_channel_check
        CHECK (channel IN ('whatsapp', 'phone', 'in_person', 'other')),
    CONSTRAINT retention_interventions_status_check
        CHECK (status IN ('planned', 'completed', 'cancelled')),
    CONSTRAINT retention_interventions_outcome_check
        CHECK (outcome IS NULL OR outcome IN ('contacted', 'no_response', 'follow_up', 'paused', 'not_interested', 'other')),
    CONSTRAINT retention_interventions_completed_at_check
        CHECK (status <> 'completed' OR completed_at IS NOT NULL)
);

CREATE INDEX idx_retention_interventions_box_status_date
    ON retention_interventions(box_id, status, created_at DESC);
CREATE INDEX idx_retention_interventions_box_student_date
    ON retention_interventions(box_id, student_id, created_at DESC);
