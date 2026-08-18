CREATE TABLE membership_plans (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    home_unit_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_key VARCHAR(100) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plans_organization_name_key_key UNIQUE (organization_id, name_key),
    CONSTRAINT membership_plans_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT membership_plans_home_unit_organization_fk
        FOREIGN KEY (home_unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plans_name_check CHECK (char_length(name) BETWEEN 1 AND 100),
    CONSTRAINT membership_plans_name_key_check CHECK (char_length(name_key) BETWEEN 1 AND 100),
    CONSTRAINT membership_plans_status_check CHECK (status IN ('active', 'archived'))
);

CREATE INDEX idx_membership_plans_home_unit_status
    ON membership_plans(home_unit_id, status);

CREATE TABLE membership_plan_versions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    price_cents INTEGER NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'BRL',
    billing_cycle VARCHAR(16) NOT NULL,
    validity_count INTEGER,
    validity_unit VARCHAR(8),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_versions_plan_number_key UNIQUE (plan_id, version_number),
    CONSTRAINT membership_plan_versions_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT membership_plan_versions_plan_organization_fk
        FOREIGN KEY (plan_id, organization_id)
        REFERENCES membership_plans(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_versions_number_check CHECK (version_number > 0),
    CONSTRAINT membership_plan_versions_revision_check CHECK (revision > 0),
    CONSTRAINT membership_plan_versions_status_check CHECK (status IN ('draft', 'published', 'retired')),
    CONSTRAINT membership_plan_versions_name_check CHECK (char_length(name) BETWEEN 1 AND 100),
    CONSTRAINT membership_plan_versions_description_check CHECK (char_length(description) <= 500),
    CONSTRAINT membership_plan_versions_price_check CHECK (price_cents > 0),
    CONSTRAINT membership_plan_versions_currency_check CHECK (currency = 'BRL'),
    CONSTRAINT membership_plan_versions_billing_cycle_check
        CHECK (billing_cycle IN ('one_time', 'monthly', 'quarterly', 'semiannual', 'annual')),
    CONSTRAINT membership_plan_versions_validity_check CHECK (
        (billing_cycle = 'one_time' AND validity_count IS NOT NULL AND validity_unit IS NOT NULL AND (
            (validity_unit = 'day' AND validity_count BETWEEN 1 AND 3650) OR
            (validity_unit = 'week' AND validity_count BETWEEN 1 AND 520) OR
            (validity_unit = 'month' AND validity_count BETWEEN 1 AND 120) OR
            (validity_unit = 'year' AND validity_count BETWEEN 1 AND 10)
        )) OR
        (billing_cycle <> 'one_time' AND validity_count IS NULL AND validity_unit IS NULL)
    ),
    CONSTRAINT membership_plan_versions_published_at_check CHECK (
        (status = 'draft' AND published_at IS NULL) OR
        (status IN ('published', 'retired') AND published_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_membership_plan_versions_one_draft
    ON membership_plan_versions(plan_id) WHERE status = 'draft';
CREATE UNIQUE INDEX idx_membership_plan_versions_one_published
    ON membership_plan_versions(plan_id) WHERE status = 'published';
CREATE INDEX idx_membership_plan_versions_plan_status
    ON membership_plan_versions(plan_id, status);

CREATE TABLE membership_plan_entitlements (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    plan_version_id UUID NOT NULL,
    modality_id UUID,
    service_offering_id UUID,
    access_type VARCHAR(32) NOT NULL,
    quantity INTEGER,
    period_unit VARCHAR(8),
    daily_limit INTEGER,
    delivery_mode VARCHAR(16),
    booking_required BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_entitlements_version_organization_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_entitlements_modality_organization_fk
        FOREIGN KEY (modality_id, organization_id)
        REFERENCES modalities(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_entitlements_service_organization_fk
        FOREIGN KEY (service_offering_id, organization_id)
        REFERENCES service_offerings(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_entitlements_target_check CHECK (
        (modality_id IS NOT NULL)::integer + (service_offering_id IS NOT NULL)::integer = 1
    ),
    CONSTRAINT membership_plan_entitlements_access_type_check
        CHECK (access_type IN ('open_access', 'sessions_per_period', 'class_pack')),
    CONSTRAINT membership_plan_entitlements_access_configuration_check CHECK (
        (access_type = 'open_access' AND quantity IS NULL AND period_unit IS NULL) OR
        (access_type = 'sessions_per_period' AND quantity BETWEEN 1 AND 10000 AND period_unit IN ('day', 'week', 'month')) OR
        (access_type = 'class_pack' AND quantity BETWEEN 1 AND 10000 AND period_unit IS NULL)
    ),
    CONSTRAINT membership_plan_entitlements_daily_limit_check
        CHECK (daily_limit IS NULL OR daily_limit BETWEEN 1 AND 10000),
    CONSTRAINT membership_plan_entitlements_delivery_mode_check
        CHECK (delivery_mode IS NULL OR delivery_mode IN ('group', 'individual')),
    CONSTRAINT membership_plan_entitlements_position_check CHECK (position >= 0),
    CONSTRAINT membership_plan_entitlements_target_unique
        UNIQUE NULLS NOT DISTINCT (plan_version_id, modality_id, service_offering_id)
);

CREATE INDEX idx_membership_plan_entitlements_version_position
    ON membership_plan_entitlements(plan_version_id, position);
CREATE INDEX idx_membership_plan_entitlements_modality
    ON membership_plan_entitlements(modality_id) WHERE modality_id IS NOT NULL;
CREATE INDEX idx_membership_plan_entitlements_service
    ON membership_plan_entitlements(service_offering_id) WHERE service_offering_id IS NOT NULL;

CREATE TABLE membership_plan_command_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    command VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    plan_id UUID NOT NULL,
    plan_version_id UUID NOT NULL,
    response_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_command_requests_scope_key
        UNIQUE (organization_id, command, idempotency_key),
    CONSTRAINT membership_plan_command_requests_plan_organization_fk
        FOREIGN KEY (plan_id, organization_id)
        REFERENCES membership_plans(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_command_requests_version_organization_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_command_requests_command_check
        CHECK (command IN ('create', 'publish', 'new_version')),
    CONSTRAINT membership_plan_command_requests_key_check
        CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'),
    CONSTRAINT membership_plan_command_requests_fingerprint_check
        CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE membership_plan_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    plan_id UUID NOT NULL,
    plan_version_id UUID,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_audit_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_audit_plan_organization_fk
        FOREIGN KEY (plan_id, organization_id)
        REFERENCES membership_plans(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_audit_version_organization_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_membership_plan_audit_organization_created_at
    ON membership_plan_audit_events(organization_id, created_at DESC);
CREATE INDEX idx_membership_plan_audit_plan_created_at
    ON membership_plan_audit_events(plan_id, created_at DESC);

CREATE OR REPLACE FUNCTION prevent_membership_plan_version_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status IN ('published', 'retired') THEN
            RAISE EXCEPTION 'published membership plan versions are immutable' USING ERRCODE = '55000';
        END IF;
        RETURN OLD;
    END IF;

    IF OLD.status = 'published' AND NEW.status = 'retired'
       AND NEW.id = OLD.id
       AND NEW.organization_id = OLD.organization_id
       AND NEW.plan_id = OLD.plan_id
       AND NEW.version_number = OLD.version_number
       AND NEW.revision = OLD.revision
       AND NEW.name = OLD.name
       AND NEW.description = OLD.description
       AND NEW.price_cents = OLD.price_cents
       AND NEW.currency = OLD.currency
       AND NEW.billing_cycle = OLD.billing_cycle
       AND NEW.validity_count IS NOT DISTINCT FROM OLD.validity_count
       AND NEW.validity_unit IS NOT DISTINCT FROM OLD.validity_unit
       AND NEW.published_at = OLD.published_at
       AND NEW.created_at = OLD.created_at THEN
        RETURN NEW;
    END IF;

    IF OLD.status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan versions are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_published_membership_entitlement_mutation()
RETURNS TRIGGER AS $$
DECLARE
    current_version_id UUID;
    current_status VARCHAR(16);
BEGIN
    current_version_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.plan_version_id ELSE NEW.plan_version_id END;
    SELECT status INTO current_status
    FROM membership_plan_versions
    WHERE id = current_version_id;

    IF current_status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan entitlements are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_membership_plan_versions
BEFORE UPDATE OR DELETE ON membership_plan_versions
FOR EACH ROW EXECUTE FUNCTION prevent_membership_plan_version_mutation();

CREATE TRIGGER protect_membership_plan_entitlements
BEFORE INSERT OR UPDATE OR DELETE ON membership_plan_entitlements
FOR EACH ROW EXECUTE FUNCTION prevent_published_membership_entitlement_mutation();

CREATE TRIGGER set_membership_plans_updated_at
BEFORE UPDATE ON membership_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_membership_plan_versions_updated_at
BEFORE UPDATE ON membership_plan_versions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_membership_plan_entitlements_updated_at
BEFORE UPDATE ON membership_plan_entitlements
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
