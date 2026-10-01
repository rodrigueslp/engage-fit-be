-- SPEC-004/B-D: provider-neutral acceptance orchestration, isolated health
-- forms, and auditable post-issuance legal lifecycle.

CREATE TABLE contract_signature_envelopes (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    contract_instance_id UUID NOT NULL,
    provider VARCHAR(48) NOT NULL,
    provider_reference VARCHAR(160),
    status VARCHAR(24) NOT NULL DEFAULT 'created',
    signer_qualification_encrypted TEXT NOT NULL,
    evidence_encrypted TEXT,
    webhook_event_id VARCHAR(160),
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id),
    UNIQUE (organization_id, provider, provider_reference),
    UNIQUE (organization_id, provider, webhook_event_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (contract_instance_id, organization_id) REFERENCES contract_instances(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_signature_envelopes_status_check CHECK (status IN ('created','pending','accepted','declined','expired','failed')),
    CONSTRAINT contract_signature_envelopes_provider_check CHECK (char_length(provider) BETWEEN 2 AND 48),
    CONSTRAINT contract_signature_envelopes_completion_check CHECK ((status = 'accepted' AND completed_at IS NOT NULL) OR (status <> 'accepted' AND completed_at IS NULL))
);
CREATE UNIQUE INDEX idx_contract_signature_one_open ON contract_signature_envelopes(contract_instance_id) WHERE status IN ('created','pending');
CREATE INDEX idx_contract_signature_status ON contract_signature_envelopes(organization_id, status, updated_at);
CREATE TRIGGER set_contract_signature_envelopes_updated_at BEFORE UPDATE ON contract_signature_envelopes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE contract_acceptance_sessions DROP CONSTRAINT contract_acceptance_sessions_adapter_check;
ALTER TABLE contract_acceptance_sessions ALTER COLUMN adapter_reference TYPE VARCHAR(160);
ALTER TABLE contract_acceptance_sessions ADD CONSTRAINT contract_acceptance_sessions_adapter_check CHECK (adapter IN ('local_simulator','external_provider'));
ALTER TABLE contract_acceptances DROP CONSTRAINT contract_acceptances_adapter_check;
ALTER TABLE contract_acceptances ALTER COLUMN adapter_reference TYPE VARCHAR(160);
ALTER TABLE contract_acceptances ADD CONSTRAINT contract_acceptances_adapter_check CHECK (adapter IN ('local_simulator','external_provider'));

CREATE TABLE contract_signature_webhook_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    envelope_id UUID NOT NULL,
    provider VARCHAR(48) NOT NULL,
    provider_event_id VARCHAR(160) NOT NULL,
    normalized_status VARCHAR(24) NOT NULL,
    evidence_encrypted TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, provider, provider_event_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (envelope_id, organization_id) REFERENCES contract_signature_envelopes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_signature_webhook_status_check CHECK (normalized_status IN ('pending','accepted','declined','expired','failed'))
);

CREATE TABLE health_form_submissions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(24) NOT NULL DEFAULT 'submitted',
    consent_version VARCHAR(32) NOT NULL,
    consent_text TEXT NOT NULL,
    consent_accepted_at TIMESTAMPTZ NOT NULL,
    consent_revoked_at TIMESTAMPTZ,
    requires_review BOOLEAN NOT NULL DEFAULT FALSE,
    answers_encrypted TEXT NOT NULL,
    retention_due_at TIMESTAMPTZ NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (enrollment_id, organization_id, person_id, unit_id) REFERENCES enrollments(id, organization_id, person_id, home_unit_id) ON DELETE RESTRICT,
    CONSTRAINT health_form_schema_check CHECK (schema_version = 1),
    CONSTRAINT health_form_status_check CHECK (status IN ('submitted','consent_revoked','expired')),
    CONSTRAINT health_form_consent_check CHECK ((status = 'consent_revoked' AND consent_revoked_at IS NOT NULL) OR (status <> 'consent_revoked' AND consent_revoked_at IS NULL)),
    CONSTRAINT health_form_retention_check CHECK (retention_due_at > consent_accepted_at)
);
CREATE UNIQUE INDEX idx_health_form_one_active_schema ON health_form_submissions(organization_id, enrollment_id, schema_version) WHERE status = 'submitted';
CREATE INDEX idx_health_form_person ON health_form_submissions(organization_id, person_id, created_at DESC);
CREATE INDEX idx_health_form_retention ON health_form_submissions(retention_due_at);
CREATE TRIGGER set_health_form_submissions_updated_at BEFORE UPDATE ON health_form_submissions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE health_form_access_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    submission_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (submission_id, organization_id) REFERENCES health_form_submissions(id, organization_id) ON DELETE CASCADE,
    CONSTRAINT health_form_access_action_check CHECK (action IN ('answers_viewed','consent_revoked','retention_purged'))
);
CREATE INDEX idx_health_form_access_submission ON health_form_access_events(submission_id, created_at);

CREATE TABLE contract_legal_actions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    source_contract_instance_id UUID NOT NULL,
    action_type VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'recorded',
    effective_on DATE NOT NULL,
    ends_on DATE,
    reason VARCHAR(500) NOT NULL,
    terms_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    financial_instruction JSONB NOT NULL DEFAULT '{}'::jsonb,
    revision INTEGER NOT NULL DEFAULT 1,
    supersedes_action_id UUID,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (enrollment_id, organization_id) REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (source_contract_instance_id, organization_id) REFERENCES contract_instances(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (supersedes_action_id, organization_id) REFERENCES contract_legal_actions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT contract_legal_action_type_check CHECK (action_type IN ('amendment','renewal','pause','resume','cancellation')),
    CONSTRAINT contract_legal_action_status_check CHECK (status IN ('recorded','superseded','voided')),
    CONSTRAINT contract_legal_action_reason_check CHECK (char_length(reason) BETWEEN 3 AND 500),
    CONSTRAINT contract_legal_action_dates_check CHECK (ends_on IS NULL OR ends_on >= effective_on)
);
CREATE UNIQUE INDEX idx_contract_legal_one_open_pause ON contract_legal_actions(enrollment_id) WHERE action_type = 'pause' AND status = 'recorded';
CREATE INDEX idx_contract_legal_enrollment ON contract_legal_actions(organization_id, enrollment_id, created_at DESC);
CREATE TRIGGER set_contract_legal_actions_updated_at BEFORE UPDATE ON contract_legal_actions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE contract_legal_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    legal_action_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(48) NOT NULL,
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (legal_action_id, organization_id) REFERENCES contract_legal_actions(id, organization_id) ON DELETE RESTRICT
);
CREATE INDEX idx_contract_legal_events_action ON contract_legal_events(legal_action_id, created_at);
