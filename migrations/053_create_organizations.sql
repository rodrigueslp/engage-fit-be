CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    status_reason TEXT NOT NULL DEFAULT '',
    status_changed_at TIMESTAMPTZ,
    status_changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT organizations_status_check CHECK (status IN ('active', 'suspended', 'archived'))
);

CREATE INDEX idx_organizations_status ON organizations(status);

ALTER TABLE boxes
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE RESTRICT;

INSERT INTO organizations (
    id,
    name,
    status,
    status_reason,
    status_changed_at,
    status_changed_by,
    created_at,
    updated_at
)
SELECT
    id,
    name,
    status,
    status_reason,
    status_changed_at,
    status_changed_by,
    created_at,
    updated_at
FROM boxes
ON CONFLICT (id) DO NOTHING;

UPDATE boxes
SET organization_id = id
WHERE organization_id IS NULL;

-- Compatibility guard for legacy fixtures and rollout code paths that still
-- create a box directly. Application repositories create the organization
-- explicitly; this trigger guarantees the invariant during the transition.
CREATE OR REPLACE FUNCTION ensure_box_organization()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.organization_id IS NULL THEN
        INSERT INTO organizations (
            id,
            name,
            status,
            status_reason,
            status_changed_at,
            status_changed_by,
            created_at,
            updated_at
        ) VALUES (
            NEW.id,
            NEW.name,
            COALESCE(NULLIF(NEW.status, ''), 'active'),
            COALESCE(NEW.status_reason, ''),
            NEW.status_changed_at,
            NEW.status_changed_by,
            NEW.created_at,
            NEW.updated_at
        )
        ON CONFLICT (id) DO NOTHING;

        NEW.organization_id := NEW.id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ensure_boxes_have_organization
BEFORE INSERT ON boxes
FOR EACH ROW
EXECUTE FUNCTION ensure_box_organization();

ALTER TABLE boxes
    ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX idx_boxes_organization_id ON boxes(organization_id);

CREATE TRIGGER set_organizations_updated_at
BEFORE UPDATE ON organizations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
