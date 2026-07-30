ALTER TABLE boxes
    ADD COLUMN contact_activation_code UUID NOT NULL DEFAULT GEN_RANDOM_UUID();

CREATE UNIQUE INDEX idx_boxes_contact_activation_code
    ON boxes(contact_activation_code);

CREATE TABLE contact_activation_requests (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    claimed_name VARCHAR(160) NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL,
    recent_checkin_date DATE,
    sender_phone VARCHAR(32) NOT NULL,
    phone VARCHAR(32) NOT NULL DEFAULT '',
    token_hash CHAR(64) NOT NULL UNIQUE,
    status VARCHAR(24) NOT NULL DEFAULT 'awaiting_message',
    consent_version VARCHAR(32) NOT NULL,
    consent_text TEXT NOT NULL,
    consented_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contact_activation_requests_source_check
        CHECK (source IN ('wellhub', 'totalpass')),
    CONSTRAINT contact_activation_requests_status_check
        CHECK (status IN ('awaiting_message', 'confirmed', 'needs_review', 'expired', 'cancelled'))
);

CREATE INDEX idx_contact_activation_requests_box_status
    ON contact_activation_requests(box_id, status, created_at DESC);

CREATE INDEX idx_contact_activation_requests_phone_sender
    ON contact_activation_requests(phone, sender_phone)
    WHERE phone <> '';

CREATE TABLE contact_consent_events (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    activation_request_id UUID REFERENCES contact_activation_requests(id) ON DELETE SET NULL,
    phone VARCHAR(32) NOT NULL,
    action VARCHAR(24) NOT NULL,
    source VARCHAR(40) NOT NULL,
    consent_version VARCHAR(32) NOT NULL DEFAULT '',
    consent_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contact_consent_events_action_check
        CHECK (action IN ('opted_in', 'opted_out'))
);

CREATE INDEX idx_contact_consent_events_box_created
    ON contact_consent_events(box_id, created_at DESC);

CREATE INDEX idx_contact_consent_events_student_created
    ON contact_consent_events(student_id, created_at DESC);
