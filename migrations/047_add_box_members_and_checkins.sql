ALTER TABLE students DROP CONSTRAINT students_source_check;
ALTER TABLE students
    ADD CONSTRAINT students_source_check
        CHECK (source IN ('wellhub', 'totalpass', 'box_member'));

ALTER TABLE checkins DROP CONSTRAINT checkins_source_check;
ALTER TABLE checkins
    ADD CONSTRAINT checkins_source_check
        CHECK (source IN ('wellhub', 'totalpass', 'box_member'));

ALTER TABLE campaign_goals DROP CONSTRAINT campaign_goals_source_check;
ALTER TABLE campaign_goals
    ADD CONSTRAINT campaign_goals_source_check
        CHECK (source IN ('wellhub', 'totalpass', 'box_member'));

ALTER TABLE contact_activation_requests DROP CONSTRAINT contact_activation_requests_source_check;
ALTER TABLE contact_activation_requests
    ADD CONSTRAINT contact_activation_requests_source_check
        CHECK (source IN ('wellhub', 'totalpass', 'box_member'));

ALTER TABLE checkins
    ALTER COLUMN import_history_id DROP NOT NULL,
    ADD COLUMN entry_method VARCHAR(24) NOT NULL DEFAULT 'import',
    ADD COLUMN recorded_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN self_checkin_session_id UUID;

ALTER TABLE checkins
    ADD CONSTRAINT checkins_entry_method_check
        CHECK (entry_method IN ('import', 'manual', 'self_service'));

CREATE TABLE self_checkin_sessions (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT self_checkin_sessions_expiry_check CHECK (expires_at > created_at)
);

ALTER TABLE checkins
    ADD CONSTRAINT checkins_self_checkin_session_fk
        FOREIGN KEY (self_checkin_session_id) REFERENCES self_checkin_sessions(id) ON DELETE SET NULL;

CREATE INDEX idx_self_checkin_sessions_box_expiry
    ON self_checkin_sessions(box_id, expires_at DESC);

CREATE UNIQUE INDEX idx_box_member_checkins_one_per_day
    ON checkins(box_id, student_id, checkin_date)
    WHERE source = 'box_member';

CREATE INDEX idx_checkins_entry_method
    ON checkins(box_id, entry_method, checkin_date DESC);
