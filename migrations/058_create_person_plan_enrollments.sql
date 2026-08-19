CREATE TABLE enrollments (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    person_id UUID NOT NULL,
    home_unit_id UUID NOT NULL,
    plan_version_id UUID NOT NULL,
    lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'active',
    starts_on DATE NOT NULL,
    valid_until DATE,
    ended_on DATE,
    ended_at TIMESTAMPTZ,
    ended_reason VARCHAR(240),
    revision INTEGER NOT NULL DEFAULT 1,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    ended_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT enrollments_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT enrollments_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollments_unit_organization_fk
        FOREIGN KEY (home_unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollments_version_organization_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollments_status_check CHECK (lifecycle_status IN ('active', 'ended')),
    CONSTRAINT enrollments_revision_check CHECK (revision > 0),
    CONSTRAINT enrollments_validity_check CHECK (valid_until IS NULL OR valid_until >= starts_on),
    CONSTRAINT enrollments_end_fields_check CHECK (
        (lifecycle_status = 'active' AND ended_on IS NULL AND ended_at IS NULL AND ended_reason IS NULL AND ended_by_user_id IS NULL) OR
        (lifecycle_status = 'ended' AND ended_on IS NOT NULL AND ended_at IS NOT NULL AND ended_reason IS NOT NULL AND char_length(ended_reason) BETWEEN 3 AND 240 AND ended_by_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_enrollments_one_open_per_person_unit
    ON enrollments(organization_id, person_id, home_unit_id)
    WHERE lifecycle_status = 'active';
CREATE INDEX idx_enrollments_person_unit_created
    ON enrollments(organization_id, person_id, home_unit_id, created_at DESC);
CREATE INDEX idx_enrollments_plan_version ON enrollments(plan_version_id);

CREATE TABLE enrollment_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    command VARCHAR(16) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    enrollment_id UUID NOT NULL,
    response_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT enrollment_command_requests_scope_key UNIQUE (organization_id, command, idempotency_key),
    CONSTRAINT enrollment_command_requests_enrollment_organization_fk
        FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_command_requests_command_check CHECK (command IN ('create', 'end')),
    CONSTRAINT enrollment_command_requests_key_check CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT enrollment_command_requests_fingerprint_check CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE enrollment_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason VARCHAR(240),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT enrollment_events_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_events_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_events_enrollment_organization_fk
        FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_events_action_check CHECK (action IN ('enrollment.created', 'enrollment.ended', 'enrollment.expiration_reconciled')),
    CONSTRAINT enrollment_events_reason_check CHECK (reason IS NULL OR char_length(reason) BETWEEN 3 AND 240)
);

CREATE INDEX idx_enrollment_events_person_created
    ON enrollment_events(organization_id, person_id, unit_id, created_at DESC);
CREATE INDEX idx_enrollment_events_enrollment_created
    ON enrollment_events(enrollment_id, created_at ASC);

CREATE TRIGGER set_enrollments_updated_at
BEFORE UPDATE ON enrollments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
