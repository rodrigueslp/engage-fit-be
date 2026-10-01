ALTER TABLE checkins DROP CONSTRAINT checkins_entry_method_check;
ALTER TABLE checkins ADD CONSTRAINT checkins_entry_method_check
    CHECK (entry_method IN ('import', 'manual', 'self_service', 'access_decision'));

CREATE TABLE attendance_access_decisions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    session_id UUID,
    enrollment_id UUID,
    reservation_id UUID,
    source VARCHAR(16) NOT NULL,
    outcome VARCHAR(16) NOT NULL,
    reason_code VARCHAR(48) NOT NULL,
    reason VARCHAR(240) NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    occurred_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT attendance_decisions_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT attendance_decisions_unit_fk FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_decisions_person_fk FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_decisions_session_fk FOREIGN KEY (session_id, organization_id, unit_id)
        REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_decisions_enrollment_fk FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_decisions_reservation_fk FOREIGN KEY (reservation_id, organization_id)
        REFERENCES reservations(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_decisions_source_check CHECK (source IN ('wellhub', 'totalpass', 'box_member')),
    CONSTRAINT attendance_decisions_outcome_check CHECK (outcome IN ('allowed', 'denied')),
    CONSTRAINT attendance_decisions_reason_check CHECK (char_length(reason) BETWEEN 3 AND 240),
    CONSTRAINT attendance_decisions_session_refs_check CHECK (
        session_id IS NOT NULL OR (reservation_id IS NULL)
    )
);

CREATE INDEX idx_attendance_decisions_unit_time
    ON attendance_access_decisions(organization_id, unit_id, occurred_at DESC);
CREATE INDEX idx_attendance_decisions_person_time
    ON attendance_access_decisions(organization_id, person_id, occurred_at DESC);

CREATE TABLE attendance_records (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    decision_id UUID NOT NULL,
    person_id UUID NOT NULL,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    session_id UUID,
    checkin_id UUID REFERENCES checkins(id) ON DELETE SET NULL,
    local_date DATE NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT attendance_records_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT attendance_records_unit_fk FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_decision_fk FOREIGN KEY (decision_id, organization_id)
        REFERENCES attendance_access_decisions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_person_fk FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_session_fk FOREIGN KEY (session_id, organization_id, unit_id)
        REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_attendance_one_open_entry_per_day
    ON attendance_records(organization_id, unit_id, person_id, local_date)
    WHERE session_id IS NULL;
CREATE UNIQUE INDEX idx_attendance_one_presence_per_session
    ON attendance_records(organization_id, unit_id, session_id, person_id)
    WHERE session_id IS NOT NULL;
CREATE INDEX idx_attendance_records_unit_date
    ON attendance_records(organization_id, unit_id, local_date DESC, occurred_at DESC);

CREATE TABLE attendance_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    decision_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT attendance_commands_scope_key UNIQUE (organization_id, idempotency_key),
    CONSTRAINT attendance_commands_decision_fk FOREIGN KEY (decision_id, organization_id)
        REFERENCES attendance_access_decisions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_commands_key_check CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT attendance_commands_fingerprint_check CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);
