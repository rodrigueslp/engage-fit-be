CREATE TABLE reservations (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    session_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    entitlement_id UUID NOT NULL,
    status VARCHAR(16) NOT NULL,
    makeup_for_reservation_id UUID,
    revision INTEGER NOT NULL DEFAULT 1,
    cancelled_reason VARCHAR(240),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT reservations_id_organization_key UNIQUE (id, organization_id),
    CONSTRAINT reservations_session_scope_fk FOREIGN KEY (session_id, organization_id, unit_id)
        REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT reservations_person_scope_fk FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservations_enrollment_scope_fk FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservations_makeup_scope_fk FOREIGN KEY (makeup_for_reservation_id, organization_id)
        REFERENCES reservations(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservations_status_check CHECK (status IN ('confirmed', 'waitlisted', 'cancelled', 'no_show')),
    CONSTRAINT reservations_revision_check CHECK (revision > 0),
    CONSTRAINT reservations_cancel_reason_check CHECK (
        (status = 'cancelled' AND cancelled_reason IS NOT NULL AND char_length(cancelled_reason) BETWEEN 3 AND 240)
        OR (status <> 'cancelled' AND cancelled_reason IS NULL)
    ),
    CONSTRAINT reservations_not_own_makeup_check CHECK (makeup_for_reservation_id IS NULL OR makeup_for_reservation_id <> id)
);

CREATE UNIQUE INDEX idx_reservations_one_active_person_session
    ON reservations(organization_id, unit_id, session_id, person_id)
    WHERE status IN ('confirmed', 'waitlisted');
CREATE UNIQUE INDEX idx_reservations_one_active_makeup
    ON reservations(makeup_for_reservation_id)
    WHERE makeup_for_reservation_id IS NOT NULL AND status <> 'cancelled';
CREATE INDEX idx_reservations_session_queue
    ON reservations(organization_id, unit_id, session_id, status, created_at);
CREATE INDEX idx_reservations_person_history
    ON reservations(organization_id, unit_id, person_id, created_at DESC);
CREATE INDEX idx_reservations_entitlement_consumption
    ON reservations(organization_id, person_id, entitlement_id, status);

CREATE TABLE reservation_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    command VARCHAR(16) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    reservation_id UUID NOT NULL,
    response_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reservation_command_scope_key UNIQUE (organization_id, command, idempotency_key),
    CONSTRAINT reservation_command_reservation_fk FOREIGN KEY (reservation_id, organization_id)
        REFERENCES reservations(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservation_command_name_check CHECK (command IN ('book', 'cancel', 'no_show')),
    CONSTRAINT reservation_command_key_check CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT reservation_command_fingerprint_check CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE reservation_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    session_id UUID NOT NULL,
    reservation_id UUID NOT NULL,
    person_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    reason VARCHAR(240),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reservation_audit_unit_fk FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservation_audit_session_fk FOREIGN KEY (session_id, organization_id, unit_id)
        REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT reservation_audit_reservation_fk FOREIGN KEY (reservation_id, organization_id)
        REFERENCES reservations(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservation_audit_person_fk FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT reservation_audit_action_check CHECK (action IN ('reservation.confirmed', 'reservation.waitlisted', 'reservation.cancelled', 'reservation.promoted', 'reservation.no_show')),
    CONSTRAINT reservation_audit_reason_check CHECK (reason IS NULL OR char_length(reason) BETWEEN 3 AND 240)
);

CREATE INDEX idx_reservation_audit_session ON reservation_audit_events(organization_id, unit_id, session_id, created_at);
CREATE INDEX idx_reservation_audit_person ON reservation_audit_events(organization_id, person_id, created_at);

CREATE TRIGGER set_reservations_updated_at BEFORE UPDATE ON reservations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
