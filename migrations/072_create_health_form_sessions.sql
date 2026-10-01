CREATE TABLE health_form_sessions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    token_hash CHAR(64) NOT NULL UNIQUE,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    submitted_at TIMESTAMPTZ,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (enrollment_id, organization_id, person_id, unit_id) REFERENCES enrollments(id, organization_id, person_id, home_unit_id) ON DELETE RESTRICT,
    CONSTRAINT health_form_session_schema_check CHECK (schema_version = 1),
    CONSTRAINT health_form_session_status_check CHECK (status IN ('active','submitted','expired','superseded')),
    CONSTRAINT health_form_session_submission_check CHECK ((status = 'submitted' AND submitted_at IS NOT NULL) OR (status <> 'submitted' AND submitted_at IS NULL)),
    CONSTRAINT health_form_session_expiry_check CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX idx_health_form_session_active
    ON health_form_sessions(organization_id, enrollment_id, schema_version)
    WHERE status = 'active';
CREATE INDEX idx_health_form_session_expiry ON health_form_sessions(expires_at) WHERE status = 'active';
CREATE TRIGGER set_health_form_sessions_updated_at BEFORE UPDATE ON health_form_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
