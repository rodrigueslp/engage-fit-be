CREATE TABLE persons (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name VARCHAR(160) NOT NULL,
    phone_e164 VARCHAR(16),
    email VARCHAR(254),
    cpf_encrypted TEXT,
    cpf_lookup_hash VARCHAR(80),
    cpf_last2 CHAR(2),
    profile_status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT persons_id_organization_id_key UNIQUE (id, organization_id),
    CONSTRAINT persons_name_check CHECK (char_length(name) BETWEEN 2 AND 160),
    CONSTRAINT persons_phone_check CHECK (phone_e164 IS NULL OR phone_e164 ~ '^\+55[0-9]{10,11}$'),
    CONSTRAINT persons_cpf_fields_check CHECK (
        (cpf_encrypted IS NULL AND cpf_lookup_hash IS NULL AND cpf_last2 IS NULL) OR
        (cpf_encrypted IS NOT NULL AND cpf_lookup_hash IS NOT NULL AND cpf_last2 IS NOT NULL)
    ),
    CONSTRAINT persons_profile_status_check CHECK (profile_status IN ('active', 'archived', 'anonymized')),
    CONSTRAINT persons_anonymized_cpf_check CHECK (
        profile_status <> 'anonymized' OR
        (cpf_encrypted IS NULL AND cpf_lookup_hash IS NULL AND cpf_last2 IS NULL)
    )
);

CREATE UNIQUE INDEX idx_persons_organization_cpf_lookup
    ON persons(organization_id, cpf_lookup_hash)
    WHERE cpf_lookup_hash IS NOT NULL;
CREATE INDEX idx_persons_organization_name ON persons(organization_id, lower(name));
CREATE INDEX idx_persons_organization_phone ON persons(organization_id, phone_e164);

CREATE TABLE person_addresses (
    id UUID PRIMARY KEY,
    person_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    address_type VARCHAR(16) NOT NULL DEFAULT 'home',
    postal_code VARCHAR(8),
    street VARCHAR(160),
    number VARCHAR(30),
    complement VARCHAR(100),
    district VARCHAR(100),
    city VARCHAR(100),
    state CHAR(2),
    country CHAR(2) NOT NULL DEFAULT 'BR',
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT person_addresses_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE CASCADE,
    CONSTRAINT person_addresses_person_type_key UNIQUE (person_id, address_type),
    CONSTRAINT person_addresses_type_check CHECK (address_type = 'home'),
    CONSTRAINT person_addresses_postal_code_check CHECK (postal_code IS NULL OR postal_code ~ '^[0-9]{8}$'),
    CONSTRAINT person_addresses_state_check CHECK (state IS NULL OR state ~ '^[A-Z]{2}$'),
    CONSTRAINT person_addresses_country_check CHECK (country ~ '^[A-Z]{2}$')
);

CREATE TABLE person_unit_relationships (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    person_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    relationship_status VARCHAR(16) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    status_reason VARCHAR(280) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT person_unit_relationships_person_unit_key UNIQUE (person_id, unit_id),
    CONSTRAINT person_unit_relationships_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_unit_relationships_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_unit_relationships_status_check CHECK (relationship_status IN ('active', 'inactive', 'blocked')),
    CONSTRAINT person_unit_relationships_period_check CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_person_relationships_unit_status
    ON person_unit_relationships(unit_id, relationship_status);

ALTER TABLE students
    ADD COLUMN person_id UUID REFERENCES persons(id) ON DELETE SET NULL;
CREATE INDEX idx_students_person_id ON students(person_id);

WITH normalized_students AS (
    SELECT
        s.id,
        b.organization_id,
        left(regexp_replace(btrim(s.name), '[[:space:]]+', ' ', 'g'), 160) AS normalized_name,
        CASE
            WHEN length(regexp_replace(COALESCE(s.phone, ''), '[^0-9]', '', 'g')) IN (10, 11)
                THEN '+55' || regexp_replace(s.phone, '[^0-9]', '', 'g')
            WHEN length(regexp_replace(COALESCE(s.phone, ''), '[^0-9]', '', 'g')) IN (12, 13)
                 AND regexp_replace(s.phone, '[^0-9]', '', 'g') LIKE '55%'
                THEN '+' || regexp_replace(s.phone, '[^0-9]', '', 'g')
            ELSE NULL
        END AS normalized_phone,
        CASE
            WHEN char_length(btrim(COALESCE(s.email, ''))) BETWEEN 3 AND 254
                 AND btrim(s.email) ~* '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
                THEN lower(btrim(s.email))
            ELSE NULL
        END AS normalized_email,
        CASE WHEN s.anonymized_at IS NULL THEN 'active' ELSE 'anonymized' END AS profile_status,
        s.created_at,
        s.updated_at,
        COALESCE(s.membership_started_at::timestamptz, s.created_at) AS relationship_started_at
    FROM students s
    JOIN boxes b ON b.id = s.box_id
)
INSERT INTO persons (
    id, organization_id, name, phone_e164, email, profile_status, created_at, updated_at
)
SELECT
    id,
    organization_id,
    CASE WHEN char_length(normalized_name) >= 2 THEN normalized_name ELSE 'Aluno' END,
    CASE WHEN profile_status = 'anonymized' THEN NULL ELSE normalized_phone END,
    CASE WHEN profile_status = 'anonymized' THEN NULL ELSE normalized_email END,
    profile_status,
    created_at,
    updated_at
FROM normalized_students;

INSERT INTO person_unit_relationships (
    id, organization_id, person_id, unit_id, relationship_status,
    started_at, created_at, updated_at
)
SELECT
    s.id,
    b.organization_id,
    s.id,
    s.box_id,
    CASE WHEN s.anonymized_at IS NULL THEN 'active' ELSE 'inactive' END,
    COALESCE(s.membership_started_at::timestamptz, s.created_at),
    s.created_at,
    s.updated_at
FROM students s
JOIN boxes b ON b.id = s.box_id;

UPDATE students SET person_id = id WHERE person_id IS NULL;

DO $$
DECLARE
    students_count BIGINT;
    linked_count BIGINT;
    persons_count BIGINT;
    relationships_count BIGINT;
BEGIN
    SELECT COUNT(*) INTO students_count FROM students;
    SELECT COUNT(*) INTO linked_count FROM students WHERE person_id IS NOT NULL;
    SELECT COUNT(*) INTO persons_count
    FROM persons p JOIN students s ON s.id = p.id AND s.person_id = p.id;
    SELECT COUNT(*) INTO relationships_count
    FROM person_unit_relationships r JOIN students s ON s.id = r.id AND s.person_id = r.person_id AND s.box_id = r.unit_id;

    IF students_count <> linked_count OR students_count <> persons_count OR students_count <> relationships_count THEN
        RAISE EXCEPTION 'people backfill reconciliation failed: students %, linked %, persons %, relationships %',
            students_count, linked_count, persons_count, relationships_count;
    END IF;
END $$;

CREATE TABLE person_creation_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint VARCHAR(80) NOT NULL,
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT person_creation_requests_scope_key UNIQUE (organization_id, unit_id, idempotency_key),
    CONSTRAINT person_creation_requests_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_creation_requests_key_check CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$')
);

CREATE TABLE people_audit_events (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT people_audit_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_people_audit_organization_created_at
    ON people_audit_events(organization_id, created_at DESC);
CREATE INDEX idx_people_audit_person_created_at
    ON people_audit_events(person_id, created_at DESC);

CREATE TRIGGER set_persons_updated_at
BEFORE UPDATE ON persons
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_person_addresses_updated_at
BEFORE UPDATE ON person_addresses
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_person_unit_relationships_updated_at
BEFORE UPDATE ON person_unit_relationships
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
