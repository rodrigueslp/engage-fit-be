ALTER TABLE boxes
    ADD CONSTRAINT boxes_id_organization_id_key UNIQUE (id, organization_id);

CREATE TABLE modalities (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name VARCHAR(80) NOT NULL,
    name_key VARCHAR(80) NOT NULL,
    color CHAR(7) NOT NULL DEFAULT '#0F766E',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT modalities_organization_name_key_key UNIQUE (organization_id, name_key),
    CONSTRAINT modalities_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT modalities_name_check CHECK (char_length(name) BETWEEN 1 AND 80),
    CONSTRAINT modalities_name_key_check CHECK (char_length(name_key) BETWEEN 1 AND 80),
    CONSTRAINT modalities_color_check CHECK (color ~ '^#[0-9A-F]{6}$'),
    CONSTRAINT modalities_status_check CHECK (status IN ('active', 'archived'))
);

CREATE TABLE unit_modalities (
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    modality_id UUID NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (unit_id, modality_id),
    CONSTRAINT unit_modalities_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT unit_modalities_modality_organization_fk
        FOREIGN KEY (modality_id, organization_id)
        REFERENCES modalities(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_unit_modalities_modality_id ON unit_modalities(modality_id);

CREATE TABLE spaces (
    id UUID PRIMARY KEY,
    unit_id UUID NOT NULL REFERENCES boxes(id) ON DELETE RESTRICT,
    name VARCHAR(80) NOT NULL,
    name_key VARCHAR(80) NOT NULL,
    default_capacity INTEGER,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT spaces_unit_name_key_key UNIQUE (unit_id, name_key),
    CONSTRAINT spaces_id_unit_id_key UNIQUE (id, unit_id),
    CONSTRAINT spaces_name_check CHECK (char_length(name) BETWEEN 1 AND 80),
    CONSTRAINT spaces_name_key_check CHECK (char_length(name_key) BETWEEN 1 AND 80),
    CONSTRAINT spaces_default_capacity_check CHECK (default_capacity BETWEEN 1 AND 10000),
    CONSTRAINT spaces_status_check CHECK (status IN ('active', 'archived'))
);

CREATE TABLE catalog_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID REFERENCES boxes(id) ON DELETE RESTRICT,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(32) NOT NULL,
    entity_id UUID NOT NULL,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT catalog_audit_entity_type_check
        CHECK (entity_type IN ('modality', 'unit_modality', 'space'))
);

CREATE INDEX idx_catalog_audit_organization_created_at
    ON catalog_audit_events(organization_id, created_at DESC);
CREATE INDEX idx_catalog_audit_unit_created_at
    ON catalog_audit_events(unit_id, created_at DESC);

CREATE TRIGGER set_modalities_updated_at
BEFORE UPDATE ON modalities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_unit_modalities_updated_at
BEFORE UPDATE ON unit_modalities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_spaces_updated_at
BEFORE UPDATE ON spaces
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
