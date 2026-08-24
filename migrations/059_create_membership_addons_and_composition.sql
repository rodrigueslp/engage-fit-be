CREATE TABLE membership_plan_version_addons (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    plan_version_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    price_delta_cents INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_version_addons_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT membership_plan_version_addons_version_org_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_version_addons_version_name_key
        UNIQUE (plan_version_id, name),
    CONSTRAINT membership_plan_version_addons_name_check
        CHECK (char_length(name) BETWEEN 1 AND 100),
    CONSTRAINT membership_plan_version_addons_description_check
        CHECK (char_length(description) <= 500),
    CONSTRAINT membership_plan_version_addons_price_delta_check
        CHECK (price_delta_cents >= 0),
    CONSTRAINT membership_plan_version_addons_position_check
        CHECK (position >= 0)
);

CREATE INDEX idx_plan_version_addons_version_pos
    ON membership_plan_version_addons(plan_version_id, position);

CREATE TABLE membership_plan_version_addon_entitlements (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    addon_id UUID NOT NULL,
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

    CONSTRAINT membership_addon_entitlements_addon_org_fk
        FOREIGN KEY (addon_id, organization_id)
        REFERENCES membership_plan_version_addons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_modality_org_fk
        FOREIGN KEY (modality_id, organization_id)
        REFERENCES modalities(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_service_org_fk
        FOREIGN KEY (service_offering_id, organization_id)
        REFERENCES service_offerings(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_target_check CHECK (
        (modality_id IS NOT NULL)::integer + (service_offering_id IS NOT NULL)::integer = 1
    ),
    CONSTRAINT membership_addon_entitlements_access_type_check
        CHECK (access_type IN ('open_access', 'sessions_per_period', 'class_pack')),
    CONSTRAINT membership_addon_entitlements_configuration_check CHECK (
        (access_type = 'open_access' AND quantity IS NULL AND period_unit IS NULL) OR
        (access_type = 'sessions_per_period' AND quantity BETWEEN 1 AND 10000 AND period_unit IN ('day', 'week', 'month')) OR
        (access_type = 'class_pack' AND quantity BETWEEN 1 AND 10000 AND period_unit IS NULL)
    ),
    CONSTRAINT membership_addon_entitlements_daily_limit_check
        CHECK (daily_limit IS NULL OR daily_limit BETWEEN 1 AND 10000),
    CONSTRAINT membership_addon_entitlements_delivery_mode_check
        CHECK (delivery_mode IS NULL OR delivery_mode IN ('group', 'individual')),
    CONSTRAINT membership_addon_entitlements_position_check
        CHECK (position >= 0),
    CONSTRAINT membership_addon_entitlements_target_unique
        UNIQUE NULLS NOT DISTINCT (addon_id, modality_id, service_offering_id)
);

CREATE INDEX idx_plan_addon_entitlements_addon_pos
    ON membership_plan_version_addon_entitlements(addon_id, position);
CREATE INDEX idx_plan_addon_entitlements_modality
    ON membership_plan_version_addon_entitlements(modality_id) WHERE modality_id IS NOT NULL;
CREATE INDEX idx_plan_addon_entitlements_service
    ON membership_plan_version_addon_entitlements(service_offering_id) WHERE service_offering_id IS NOT NULL;

CREATE OR REPLACE FUNCTION prevent_published_plan_addon_mutation()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(16);
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        SELECT status INTO v_status
        FROM membership_plan_versions
        WHERE id = OLD.plan_version_id;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        SELECT status INTO v_status
        FROM membership_plan_versions
        WHERE id = NEW.plan_version_id;
    END IF;

    IF v_status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan version addons are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_membership_plan_version_addons
BEFORE INSERT OR UPDATE OR DELETE ON membership_plan_version_addons
FOR EACH ROW EXECUTE FUNCTION prevent_published_plan_addon_mutation();

CREATE OR REPLACE FUNCTION prevent_published_plan_addon_entitlement_mutation()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(16);
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        SELECT v.status INTO v_status
        FROM membership_plan_version_addons a
        JOIN membership_plan_versions v ON v.id = a.plan_version_id
        WHERE a.id = OLD.addon_id;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        SELECT v.status INTO v_status
        FROM membership_plan_version_addons a
        JOIN membership_plan_versions v ON v.id = a.plan_version_id
        WHERE a.id = NEW.addon_id;
    END IF;

    IF v_status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan addon entitlements are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_membership_plan_addon_entitlements
BEFORE INSERT OR UPDATE OR DELETE ON membership_plan_version_addon_entitlements
FOR EACH ROW EXECUTE FUNCTION prevent_published_plan_addon_entitlement_mutation();

CREATE TRIGGER set_membership_plan_version_addons_updated_at
BEFORE UPDATE ON membership_plan_version_addons
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_membership_plan_version_addon_entitlements_updated_at
BEFORE UPDATE ON membership_plan_version_addon_entitlements
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE enrollment_addons (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    plan_version_addon_id UUID NOT NULL,
    addon_name VARCHAR(100) NOT NULL,
    price_delta_cents INTEGER NOT NULL,
    entitlement_summary VARCHAR(240) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT enrollment_addons_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT enrollment_addons_enrollment_org_fk
        FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_addons_addon_org_fk
        FOREIGN KEY (plan_version_addon_id, organization_id)
        REFERENCES membership_plan_version_addons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_addons_enrollment_addon_key
        UNIQUE (enrollment_id, plan_version_addon_id),
    CONSTRAINT enrollment_addons_name_check
        CHECK (char_length(addon_name) BETWEEN 1 AND 100),
    CONSTRAINT enrollment_addons_price_delta_check
        CHECK (price_delta_cents >= 0),
    CONSTRAINT enrollment_addons_summary_check
        CHECK (char_length(entitlement_summary) <= 240)
);

CREATE INDEX idx_enrollment_addons_enrollment
    ON enrollment_addons(enrollment_id);

CREATE OR REPLACE FUNCTION prevent_enrollment_addon_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        RAISE EXCEPTION 'enrollment addons are immutable once created' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_enrollment_addons
BEFORE UPDATE OR DELETE ON enrollment_addons
FOR EACH ROW EXECUTE FUNCTION prevent_enrollment_addon_mutation();
