CREATE TABLE service_offerings (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name VARCHAR(80) NOT NULL,
    name_key VARCHAR(80) NOT NULL,
    description VARCHAR(280) NOT NULL DEFAULT '',
    color CHAR(7) NOT NULL DEFAULT '#0F766E',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT service_offerings_organization_name_key_key UNIQUE (organization_id, name_key),
    CONSTRAINT service_offerings_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT service_offerings_name_check CHECK (char_length(name) BETWEEN 1 AND 80),
    CONSTRAINT service_offerings_name_key_check CHECK (char_length(name_key) BETWEEN 1 AND 80),
    CONSTRAINT service_offerings_description_check CHECK (char_length(description) <= 280),
    CONSTRAINT service_offerings_color_check CHECK (color ~ '^#[0-9A-F]{6}$'),
    CONSTRAINT service_offerings_status_check CHECK (status IN ('active', 'archived'))
);

CREATE TABLE unit_service_offerings (
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    service_offering_id UUID NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (unit_id, service_offering_id),
    CONSTRAINT unit_service_offerings_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT unit_service_offerings_service_organization_fk
        FOREIGN KEY (service_offering_id, organization_id)
        REFERENCES service_offerings(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_unit_service_offerings_service_id
    ON unit_service_offerings(service_offering_id);

ALTER TABLE catalog_audit_events
    DROP CONSTRAINT catalog_audit_entity_type_check,
    ADD CONSTRAINT catalog_audit_entity_type_check
        CHECK (entity_type IN (
            'modality', 'unit_modality', 'space',
            'service_offering', 'unit_service_offering'
        ));

CREATE TRIGGER set_service_offerings_updated_at
BEFORE UPDATE ON service_offerings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_unit_service_offerings_updated_at
BEFORE UPDATE ON unit_service_offerings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
