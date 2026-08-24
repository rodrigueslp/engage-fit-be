CREATE TABLE scheduling_professionals (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    display_name VARCHAR(100) NOT NULL,
    display_name_key VARCHAR(100) NOT NULL,
    linked_user_id UUID,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    revision INTEGER NOT NULL DEFAULT 1,
    anonymized_at TIMESTAMPTZ,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NOT NULL,
    archived_at TIMESTAMPTZ,
    archived_by_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT scheduling_professionals_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT scheduling_professionals_name_check CHECK (char_length(display_name) BETWEEN 1 AND 100),
    CONSTRAINT scheduling_professionals_name_key_check CHECK (char_length(display_name_key) BETWEEN 1 AND 100),
    CONSTRAINT scheduling_professionals_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT scheduling_professionals_revision_check CHECK (revision > 0),
    CONSTRAINT scheduling_professionals_archive_fields_check CHECK (
        (status = 'active' AND archived_at IS NULL AND archived_by_user_id IS NULL) OR
        (status = 'archived' AND archived_at IS NOT NULL AND archived_by_user_id IS NOT NULL)
    ),
    CONSTRAINT scheduling_professionals_anonymized_fields_check CHECK (
        (anonymized_at IS NULL) OR
        (anonymized_at IS NOT NULL AND status = 'archived' AND linked_user_id IS NULL AND display_name = 'Profissional anonimizado')
    )
);

CREATE UNIQUE INDEX idx_scheduling_professionals_org_linked_user
    ON scheduling_professionals(organization_id, linked_user_id)
    WHERE linked_user_id IS NOT NULL;

CREATE INDEX idx_scheduling_professionals_org_status_name
    ON scheduling_professionals(organization_id, status, display_name_key, id);

CREATE TABLE scheduling_professional_units (
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    professional_id UUID NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT scheduling_professional_units_pkey PRIMARY KEY (unit_id, professional_id),
    CONSTRAINT scheduling_professional_units_professional_fk
        FOREIGN KEY (professional_id, organization_id)
        REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_professional_units_unit_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_scheduling_professional_units_org_unit_active
    ON scheduling_professional_units(organization_id, unit_id, active);

CREATE TABLE scheduling_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    command VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    professional_id UUID NOT NULL,
    response_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT scheduling_command_requests_scope_key
        UNIQUE (organization_id, command, idempotency_key),
    CONSTRAINT scheduling_command_requests_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_command_requests_professional_organization_fk
        FOREIGN KEY (professional_id, organization_id)
        REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_command_requests_command_check
        CHECK (command IN ('professional_create', 'professional_update', 'professional_archive', 'professional_reactivate', 'professional_anonymize')),
    CONSTRAINT scheduling_command_requests_key_check
        CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT scheduling_command_requests_fingerprint_check
        CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE scheduling_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(32) NOT NULL DEFAULT 'professional',
    entity_id UUID NOT NULL,
    revision INTEGER NOT NULL,
    changed_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason VARCHAR(240),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT scheduling_audit_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_audit_professional_organization_fk
        FOREIGN KEY (entity_id, organization_id)
        REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_scheduling_audit_events_organization_created_at
    ON scheduling_audit_events(organization_id, created_at DESC);
CREATE INDEX idx_scheduling_audit_events_entity_created_at
    ON scheduling_audit_events(entity_id, created_at DESC);

CREATE TRIGGER set_scheduling_professionals_updated_at
BEFORE UPDATE ON scheduling_professionals
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_scheduling_professional_units_updated_at
BEFORE UPDATE ON scheduling_professional_units
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
