ALTER TABLE boxes
    ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'America/Sao_Paulo',
    ADD COLUMN name_key VARCHAR(255);

UPDATE boxes
SET name_key = LOWER(REGEXP_REPLACE(BTRIM(name), '[[:space:]]+', ' ', 'g'))
WHERE name_key IS NULL;

ALTER TABLE boxes
    ALTER COLUMN name_key SET NOT NULL,
    ADD CONSTRAINT boxes_timezone_not_blank CHECK (BTRIM(timezone) <> ''),
    ADD CONSTRAINT boxes_name_key_not_blank CHECK (BTRIM(name_key) <> ''),
    ADD CONSTRAINT boxes_organization_name_key_key UNIQUE (organization_id, name_key);

CREATE OR REPLACE FUNCTION set_box_unit_defaults()
RETURNS TRIGGER AS $$
BEGIN
    NEW.name := REGEXP_REPLACE(BTRIM(NEW.name), '[[:space:]]+', ' ', 'g');
    NEW.name_key := LOWER(NEW.name);
    IF NEW.timezone IS NULL OR BTRIM(NEW.timezone) = '' THEN
        NEW.timezone := 'America/Sao_Paulo';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_boxes_unit_defaults
BEFORE INSERT OR UPDATE OF name, timezone ON boxes
FOR EACH ROW
EXECUTE FUNCTION set_box_unit_defaults();

CREATE TABLE organization_user_memberships (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, organization_id),
    CONSTRAINT organization_user_memberships_role_check CHECK (role IN ('OWNER', 'COACH'))
);

CREATE INDEX idx_organization_user_memberships_org_active
    ON organization_user_memberships(organization_id, active);

CREATE TABLE user_unit_access (
    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, unit_id),
    CONSTRAINT user_unit_access_membership_fk
        FOREIGN KEY (user_id, organization_id)
        REFERENCES organization_user_memberships(user_id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT user_unit_access_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_user_unit_access_user_active
    ON user_unit_access(user_id, active);
CREATE INDEX idx_user_unit_access_org_unit_active
    ON user_unit_access(organization_id, unit_id, active);

CREATE TABLE organization_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(40) NOT NULL,
    entity_id UUID NOT NULL,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT organization_audit_events_entity_type_check
        CHECK (entity_type IN ('unit', 'user_unit_access')),
    CONSTRAINT organization_audit_events_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_organization_audit_events_org_created
    ON organization_audit_events(organization_id, created_at DESC);

INSERT INTO organization_user_memberships (
    organization_id,
    user_id,
    role,
    active,
    created_at,
    updated_at
)
SELECT
    b.organization_id,
    u.id,
    u.role,
    u.active,
    u.created_at,
    u.updated_at
FROM users u
JOIN boxes b ON b.id = u.box_id
WHERE u.role IN ('OWNER', 'COACH')
ON CONFLICT (user_id, organization_id) DO NOTHING;

INSERT INTO user_unit_access (
    organization_id,
    user_id,
    unit_id,
    active,
    granted_by_user_id,
    created_at,
    updated_at
)
SELECT
    b.organization_id,
    u.id,
    b.id,
    TRUE,
    CASE WHEN u.role = 'OWNER' THEN u.id ELSE NULL END,
    u.created_at,
    u.updated_at
FROM users u
JOIN boxes b ON b.id = u.box_id
WHERE u.role IN ('OWNER', 'COACH')
ON CONFLICT (user_id, unit_id) DO NOTHING;

CREATE OR REPLACE FUNCTION sync_user_organization_access()
RETURNS TRIGGER AS $$
DECLARE
    target_organization_id UUID;
BEGIN
    IF NEW.box_id IS NULL OR NEW.role NOT IN ('OWNER', 'COACH') THEN
        RETURN NEW;
    END IF;

    SELECT organization_id
    INTO target_organization_id
    FROM boxes
    WHERE id = NEW.box_id;

    IF target_organization_id IS NULL THEN
        RETURN NEW;
    END IF;

    INSERT INTO organization_user_memberships (
        organization_id, user_id, role, active, created_at, updated_at
    ) VALUES (
        target_organization_id, NEW.id, NEW.role, NEW.active,
        COALESCE(NEW.created_at, NOW()), COALESCE(NEW.updated_at, NOW())
    )
    ON CONFLICT (user_id, organization_id) DO UPDATE
    SET role = EXCLUDED.role,
        active = EXCLUDED.active,
        updated_at = EXCLUDED.updated_at;

    IF TG_OP = 'INSERT' OR NEW.box_id IS DISTINCT FROM OLD.box_id OR NEW.role IS DISTINCT FROM OLD.role THEN
        INSERT INTO user_unit_access (
            organization_id, user_id, unit_id, active, granted_by_user_id,
            created_at, updated_at
        ) VALUES (
            target_organization_id, NEW.id, NEW.box_id, TRUE,
            CASE WHEN NEW.role = 'OWNER' THEN NEW.id ELSE NULL END,
            COALESCE(NEW.created_at, NOW()), COALESCE(NEW.updated_at, NOW())
        )
        ON CONFLICT (user_id, unit_id) DO UPDATE
        SET active = TRUE,
            updated_at = EXCLUDED.updated_at;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sync_users_to_organization_access
AFTER INSERT OR UPDATE OF box_id, role, active ON users
FOR EACH ROW
EXECUTE FUNCTION sync_user_organization_access();

CREATE TRIGGER set_organization_user_memberships_updated_at
BEFORE UPDATE ON organization_user_memberships
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_user_unit_access_updated_at
BEFORE UPDATE ON user_unit_access
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
