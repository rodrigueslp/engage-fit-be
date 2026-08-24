-- SPEC-004/A: Contract Foundation and Enrollment Issuance
-- Add candidate composite key to enrollments
ALTER TABLE enrollments ADD CONSTRAINT enrollments_id_org_person_unit_key UNIQUE (id, organization_id, person_id, home_unit_id);

-- Contract Templates
CREATE TABLE contract_templates (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    home_unit_id UUID NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(160) NOT NULL,
    name_key VARCHAR(160) NOT NULL,
    contract_kind VARCHAR(32) NOT NULL DEFAULT 'membership_terms',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_templates_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT contract_templates_unit_org_fk
        FOREIGN KEY (home_unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_templates_org_name_key UNIQUE (organization_id, name_key),
    CONSTRAINT contract_templates_kind_check CHECK (contract_kind = 'membership_terms'),
    CONSTRAINT contract_templates_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT contract_templates_name_check CHECK (char_length(name) BETWEEN 1 AND 160),
    CONSTRAINT contract_templates_name_key_check CHECK (char_length(name_key) BETWEEN 1 AND 160),
    CONSTRAINT contract_templates_code_check CHECK (char_length(code) BETWEEN 1 AND 64)
);

CREATE INDEX idx_contract_templates_org_status
    ON contract_templates(organization_id, status);
CREATE INDEX idx_contract_templates_home_unit
    ON contract_templates(home_unit_id);

-- Contract Template Versions
CREATE TABLE contract_template_versions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    template_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    title VARCHAR(160) NOT NULL,
    body_markdown TEXT NOT NULL,
    acceptance_statement VARCHAR(1000) NOT NULL,
    placeholder_schema_version INTEGER NOT NULL DEFAULT 1,
    renderer_version VARCHAR(32) NOT NULL DEFAULT 'markdown-v1',
    content_sha256 CHAR(64),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_template_versions_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT contract_template_versions_template_org_fk
        FOREIGN KEY (template_id, organization_id)
        REFERENCES contract_templates(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_template_versions_template_version_key UNIQUE (template_id, version_number),
    CONSTRAINT contract_template_versions_version_check CHECK (version_number >= 1),
    CONSTRAINT contract_template_versions_revision_check CHECK (revision >= 1),
    CONSTRAINT contract_template_versions_status_check CHECK (status IN ('draft', 'published', 'retired')),
    CONSTRAINT contract_template_versions_title_check CHECK (char_length(title) BETWEEN 1 AND 160),
    CONSTRAINT contract_template_versions_body_check CHECK (char_length(body_markdown) BETWEEN 1 AND 50000),
    CONSTRAINT contract_template_versions_acceptance_check CHECK (char_length(acceptance_statement) BETWEEN 10 AND 1000),
    CONSTRAINT contract_template_versions_schema_check CHECK (placeholder_schema_version = 1),
    CONSTRAINT contract_template_versions_renderer_check CHECK (renderer_version = 'markdown-v1'),
    CONSTRAINT contract_template_versions_published_fields_check CHECK (
        (status = 'draft' AND published_at IS NULL AND content_sha256 IS NULL) OR
        (status IN ('published', 'retired') AND published_at IS NOT NULL AND content_sha256 IS NOT NULL AND content_sha256 ~ '^[0-9a-f]{64}$')
    )
);

CREATE UNIQUE INDEX idx_contract_template_versions_one_draft
    ON contract_template_versions(template_id) WHERE status = 'draft';
CREATE UNIQUE INDEX idx_contract_template_versions_one_published
    ON contract_template_versions(template_id) WHERE status = 'published';
CREATE INDEX idx_contract_template_versions_template_vnum
    ON contract_template_versions(template_id, version_number DESC);

-- Contract Instances
CREATE TABLE contract_instances (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    template_version_id UUID NOT NULL,
    contract_kind VARCHAR(32) NOT NULL DEFAULT 'membership_terms',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    revision INTEGER NOT NULL DEFAULT 1,
    snapshot_schema_version INTEGER NOT NULL DEFAULT 1,
    commercial_snapshot JSONB NOT NULL,
    commercial_snapshot_sha256 CHAR(64) NOT NULL,
    party_snapshot JSONB NOT NULL,
    document_title VARCHAR(160) NOT NULL,
    document_body_markdown TEXT NOT NULL,
    document_sha256 CHAR(64) NOT NULL,
    acceptance_statement VARCHAR(1000) NOT NULL,
    acceptance_statement_version INTEGER NOT NULL DEFAULT 1,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    issued_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    discarded_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    issued_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    discarded_at TIMESTAMPTZ,
    retention_class VARCHAR(32) NOT NULL DEFAULT 'local_simulator',
    retention_due_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_instances_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT contract_instances_template_version_org_fk
        FOREIGN KEY (template_version_id, organization_id)
        REFERENCES contract_template_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_instances_enrollment_composite_fk
        FOREIGN KEY (enrollment_id, organization_id, person_id, unit_id)
        REFERENCES enrollments(id, organization_id, person_id, home_unit_id) ON DELETE RESTRICT,
    CONSTRAINT contract_instances_status_check CHECK (status IN ('draft', 'pending_acceptance', 'accepted', 'discarded')),
    CONSTRAINT contract_instances_kind_check CHECK (contract_kind = 'membership_terms'),
    CONSTRAINT contract_instances_revision_check CHECK (revision >= 1),
    CONSTRAINT contract_instances_retention_check CHECK (retention_class IN ('local_simulator', 'production_contract')),
    CONSTRAINT contract_instances_snapshot_schema_check CHECK (snapshot_schema_version = 1),
    CONSTRAINT contract_instances_hashes_check CHECK (
        commercial_snapshot_sha256 ~ '^[0-9a-f]{64}$' AND document_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT contract_instances_state_fields_check CHECK (
        (status = 'draft' AND issued_at IS NULL AND issued_by_user_id IS NULL AND accepted_at IS NULL AND discarded_at IS NULL AND discarded_by_user_id IS NULL) OR
        (status = 'pending_acceptance' AND issued_at IS NOT NULL AND issued_by_user_id IS NOT NULL AND accepted_at IS NULL AND discarded_at IS NULL AND discarded_by_user_id IS NULL) OR
        (status = 'accepted' AND issued_at IS NOT NULL AND issued_by_user_id IS NOT NULL AND accepted_at IS NOT NULL AND discarded_at IS NULL AND discarded_by_user_id IS NULL) OR
        (status = 'discarded' AND issued_at IS NULL AND issued_by_user_id IS NULL AND accepted_at IS NULL AND discarded_at IS NOT NULL AND discarded_by_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_contract_instances_one_active_per_enrollment
    ON contract_instances(organization_id, enrollment_id)
    WHERE status <> 'discarded';
CREATE INDEX idx_contract_instances_person_unit
    ON contract_instances(organization_id, person_id, unit_id, created_at DESC);
CREATE INDEX idx_contract_instances_enrollment
    ON contract_instances(enrollment_id);
CREATE INDEX idx_contract_instances_retention
    ON contract_instances(retention_class, retention_due_at)
    WHERE retention_due_at IS NOT NULL;

-- Contract Acceptance Sessions
CREATE TABLE contract_acceptance_sessions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    contract_instance_id UUID NOT NULL,
    adapter VARCHAR(32) NOT NULL DEFAULT 'local_simulator',
    adapter_reference VARCHAR(64) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_acceptance_sessions_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT contract_acceptance_sessions_instance_org_fk
        FOREIGN KEY (contract_instance_id, organization_id)
        REFERENCES contract_instances(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_acceptance_sessions_adapter_check CHECK (adapter IN ('local_simulator')),
    CONSTRAINT contract_acceptance_sessions_status_check CHECK (status IN ('active', 'used', 'expired', 'superseded')),
    CONSTRAINT contract_acceptance_sessions_token_hash_check CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT contract_acceptance_sessions_status_fields_check CHECK (
        (status = 'used' AND used_at IS NOT NULL) OR (status <> 'used' AND used_at IS NULL)
    )
);

CREATE UNIQUE INDEX idx_contract_acceptance_sessions_one_active
    ON contract_acceptance_sessions(contract_instance_id) WHERE status = 'active';
CREATE UNIQUE INDEX idx_contract_acceptance_sessions_token_hash
    ON contract_acceptance_sessions(token_hash);

-- Contract Acceptances
CREATE TABLE contract_acceptances (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    contract_instance_id UUID NOT NULL,
    acceptance_session_id UUID NOT NULL,
    adapter VARCHAR(32) NOT NULL DEFAULT 'local_simulator',
    adapter_reference VARCHAR(64) NOT NULL,
    evidence_schema_version INTEGER NOT NULL DEFAULT 1,
    person_id UUID NOT NULL,
    document_sha256 CHAR(64) NOT NULL,
    acceptance_statement_version INTEGER NOT NULL DEFAULT 1,
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_acceptances_instance_org_fk
        FOREIGN KEY (contract_instance_id, organization_id)
        REFERENCES contract_instances(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_acceptances_session_org_fk
        FOREIGN KEY (acceptance_session_id, organization_id)
        REFERENCES contract_acceptance_sessions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_acceptances_person_org_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_acceptances_instance_unique UNIQUE (contract_instance_id),
    CONSTRAINT contract_acceptances_session_unique UNIQUE (acceptance_session_id),
    CONSTRAINT contract_acceptances_adapter_check CHECK (adapter IN ('local_simulator')),
    CONSTRAINT contract_acceptances_evidence_schema_check CHECK (evidence_schema_version = 1),
    CONSTRAINT contract_acceptances_doc_hash_check CHECK (document_sha256 ~ '^[0-9a-f]{64}$')
);

-- Contract Command Requests (Idempotency)
CREATE TABLE contract_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    command VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    result_id UUID,
    response_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_command_requests_scope_key UNIQUE (organization_id, command, idempotency_key),
    CONSTRAINT contract_command_requests_key_check CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT contract_command_requests_fingerprint_check CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
    CONSTRAINT contract_command_requests_command_check CHECK (
        command IN (
            'template_create',
            'template_publish',
            'template_new_version',
            'instance_create',
            'instance_issue',
            'instance_discard',
            'acceptance_session_create',
            'accept'
        )
    )
);

-- Contract Instance Events
CREATE TABLE contract_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    contract_instance_id UUID NOT NULL,
    actor_type VARCHAR(32) NOT NULL,
    actor_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_events_unit_org_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_events_person_org_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_events_enrollment_org_fk
        FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_events_instance_org_fk
        FOREIGN KEY (contract_instance_id, organization_id)
        REFERENCES contract_instances(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_events_actor_check CHECK (actor_type IN ('owner', 'acceptance_subject', 'system')),
    CONSTRAINT contract_events_action_check CHECK (
        action IN (
            'contract.draft_created',
            'contract.draft_discarded',
            'contract.issued',
            'contract.acceptance_session_created',
            'contract.acceptance_session_superseded',
            'contract.accepted'
        )
    )
);

CREATE INDEX idx_contract_events_instance_created
    ON contract_events(contract_instance_id, created_at ASC);
CREATE INDEX idx_contract_events_person_created
    ON contract_events(organization_id, person_id, created_at DESC);

-- Contract Template Audit Events
CREATE TABLE contract_template_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    template_id UUID NOT NULL,
    template_version_id UUID,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contract_template_audit_events_unit_org_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_template_audit_events_template_org_fk
        FOREIGN KEY (template_id, organization_id)
        REFERENCES contract_templates(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_template_audit_events_action_check CHECK (
        action IN (
            'template.draft_created',
            'template.draft_updated',
            'template.published',
            'template.new_version_created',
            'template.archived',
            'template.reactivated'
        )
    )
);

CREATE INDEX idx_contract_template_audit_template_created
    ON contract_template_audit_events(template_id, created_at DESC);

-- Triggers for updated_at
CREATE TRIGGER set_contract_templates_updated_at
BEFORE UPDATE ON contract_templates
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_contract_template_versions_updated_at
BEFORE UPDATE ON contract_template_versions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_contract_instances_updated_at
BEFORE UPDATE ON contract_instances
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_contract_acceptance_sessions_updated_at
BEFORE UPDATE ON contract_acceptance_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Immutability trigger for contract_template_versions
CREATE OR REPLACE FUNCTION prevent_published_contract_template_version_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status IN ('published', 'retired') THEN
            RAISE EXCEPTION 'cannot delete published or retired contract template version' USING ERRCODE = '55000';
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status IN ('published', 'retired') THEN
            IF OLD.status = 'published' AND NEW.status = 'retired' AND
               OLD.title = NEW.title AND
               OLD.body_markdown = NEW.body_markdown AND
               OLD.acceptance_statement = NEW.acceptance_statement AND
               OLD.content_sha256 = NEW.content_sha256 AND
               OLD.version_number = NEW.version_number AND
               OLD.organization_id = NEW.organization_id AND
               OLD.template_id = NEW.template_id THEN
                RETURN NEW;
            END IF;
            RAISE EXCEPTION 'cannot modify published or retired contract template version' USING ERRCODE = '55000';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_published_contract_template_version_mutation
BEFORE UPDATE OR DELETE ON contract_template_versions
FOR EACH ROW EXECUTE FUNCTION prevent_published_contract_template_version_mutation();

-- Immutability trigger for contract_instances
CREATE OR REPLACE FUNCTION prevent_contract_instance_immutable_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.retention_class <> 'local_simulator' THEN
            RAISE EXCEPTION 'cannot delete non-simulator contract instances' USING ERRCODE = '55000';
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.id <> NEW.id OR
           OLD.organization_id <> NEW.organization_id OR
           OLD.unit_id <> NEW.unit_id OR
           OLD.person_id <> NEW.person_id OR
           OLD.enrollment_id <> NEW.enrollment_id OR
           OLD.template_version_id <> NEW.template_version_id OR
           OLD.contract_kind <> NEW.contract_kind OR
           OLD.snapshot_schema_version <> NEW.snapshot_schema_version OR
           OLD.commercial_snapshot::text <> NEW.commercial_snapshot::text OR
           OLD.commercial_snapshot_sha256 <> NEW.commercial_snapshot_sha256 OR
           OLD.party_snapshot::text <> NEW.party_snapshot::text OR
           OLD.document_title <> NEW.document_title OR
           OLD.document_body_markdown <> NEW.document_body_markdown OR
           OLD.document_sha256 <> NEW.document_sha256 OR
           OLD.acceptance_statement <> NEW.acceptance_statement OR
           OLD.acceptance_statement_version <> NEW.acceptance_statement_version OR
           OLD.created_by_user_id <> NEW.created_by_user_id OR
           OLD.created_at <> NEW.created_at OR
           OLD.retention_class <> NEW.retention_class THEN
            RAISE EXCEPTION 'cannot modify immutable contract instance fields' USING ERRCODE = '55000';
        END IF;

        IF OLD.status = 'draft' THEN
            IF NEW.status NOT IN ('draft', 'pending_acceptance', 'discarded') THEN
                RAISE EXCEPTION 'invalid state transition from draft' USING ERRCODE = '55000';
            END IF;
        ELSIF OLD.status = 'pending_acceptance' THEN
            IF NEW.status NOT IN ('pending_acceptance', 'accepted') THEN
                RAISE EXCEPTION 'invalid state transition from pending_acceptance' USING ERRCODE = '55000';
            END IF;
        ELSIF OLD.status IN ('accepted', 'discarded') THEN
            IF NEW.status <> OLD.status THEN
                RAISE EXCEPTION 'cannot transition from terminal contract status %', OLD.status USING ERRCODE = '55000';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_contract_instance_immutable_mutation
BEFORE UPDATE OR DELETE ON contract_instances
FOR EACH ROW EXECUTE FUNCTION prevent_contract_instance_immutable_mutation();
