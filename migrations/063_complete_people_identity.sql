ALTER TABLE persons
    ADD COLUMN preferred_name VARCHAR(160),
    ADD COLUMN merged_into_person_id UUID REFERENCES persons(id) ON DELETE RESTRICT,
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;

ALTER TABLE persons
    ADD CONSTRAINT persons_preferred_name_check CHECK (preferred_name IS NULL OR char_length(preferred_name) BETWEEN 2 AND 160),
    ADD CONSTRAINT persons_revision_check CHECK (revision >= 1),
    ADD CONSTRAINT persons_merge_not_self_check CHECK (merged_into_person_id IS NULL OR merged_into_person_id <> id);

CREATE INDEX idx_persons_merged_into ON persons(merged_into_person_id)
    WHERE merged_into_person_id IS NOT NULL;

CREATE TABLE person_external_identities (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    person_id UUID NOT NULL,
    unit_id UUID,
    provider VARCHAR(24) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    verified_at TIMESTAMPTZ,
    created_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT person_external_identities_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_external_identities_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id)
        REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_external_identities_provider_check CHECK (provider IN ('wellhub', 'totalpass')),
    CONSTRAINT person_external_identities_external_id_check CHECK (char_length(btrim(external_id)) BETWEEN 1 AND 255),
    CONSTRAINT person_external_identities_provider_external_key UNIQUE (organization_id, provider, external_id)
);

CREATE INDEX idx_person_external_identities_person
    ON person_external_identities(organization_id, person_id, provider);

CREATE TABLE person_athlete_accounts (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    person_id UUID NOT NULL,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    link_method VARCHAR(40) NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT person_athlete_accounts_person_organization_fk
        FOREIGN KEY (person_id, organization_id)
        REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_athlete_accounts_person_key UNIQUE (organization_id, person_id),
    CONSTRAINT person_athlete_accounts_account_key UNIQUE (organization_id, athlete_account_id),
    CONSTRAINT person_athlete_accounts_method_check CHECK (link_method IN ('confirmed_student_links', 'manual_review'))
);

CREATE TABLE person_identity_reviews (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    target_person_id UUID NOT NULL,
    source_person_id UUID NOT NULL,
    reason VARCHAR(40) NOT NULL,
    provider VARCHAR(24),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    resolution VARCHAR(16),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT person_identity_reviews_unit_organization_fk
        FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_identity_reviews_target_fk
        FOREIGN KEY (target_person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_identity_reviews_source_fk
        FOREIGN KEY (source_person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT person_identity_reviews_distinct_check CHECK (target_person_id <> source_person_id),
    CONSTRAINT person_identity_reviews_reason_check CHECK (reason IN ('external_identity_conflict', 'athlete_account_conflict', 'manual_duplicate')),
    CONSTRAINT person_identity_reviews_provider_check CHECK (provider IS NULL OR provider IN ('wellhub', 'totalpass')),
    CONSTRAINT person_identity_reviews_status_check CHECK (status IN ('pending', 'resolved')),
    CONSTRAINT person_identity_reviews_resolution_check CHECK (
        (status = 'pending' AND resolution IS NULL AND resolved_at IS NULL AND resolved_by_user_id IS NULL) OR
        (status = 'resolved' AND resolution IN ('merged', 'dismissed') AND resolved_at IS NOT NULL AND resolved_by_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_person_identity_reviews_pending_pair
    ON person_identity_reviews(organization_id, target_person_id, source_person_id, reason, COALESCE(provider, ''))
    WHERE status = 'pending';
CREATE INDEX idx_person_identity_reviews_unit_status
    ON person_identity_reviews(unit_id, status, created_at DESC);

-- Existing provider students become verified identities. If the same provider
-- identifier appears more than once in an organization, the oldest profile is
-- retained and the remaining candidates are sent to explicit review.
WITH ranked AS (
    SELECT s.id AS student_id, s.person_id, s.box_id AS unit_id, b.organization_id,
           s.source AS provider, s.external_id, s.created_at,
           row_number() OVER (PARTITION BY b.organization_id, s.source, s.external_id ORDER BY s.created_at, s.id) AS position
    FROM students s
    JOIN boxes b ON b.id = s.box_id
    WHERE s.person_id IS NOT NULL
      AND s.source IN ('wellhub', 'totalpass')
      AND btrim(COALESCE(s.external_id, '')) <> ''
)
INSERT INTO person_external_identities (
    id, organization_id, person_id, unit_id, provider, external_id, verified_at, created_at, updated_at
)
SELECT student_id, organization_id, person_id, unit_id, provider, external_id, created_at, created_at, created_at
FROM ranked WHERE position = 1;

WITH ranked AS (
    SELECT s.id AS student_id, s.person_id, s.box_id AS unit_id, b.organization_id,
           s.source AS provider, s.external_id, s.created_at,
           first_value(s.person_id) OVER (PARTITION BY b.organization_id, s.source, s.external_id ORDER BY s.created_at, s.id) AS target_person_id,
           row_number() OVER (PARTITION BY b.organization_id, s.source, s.external_id ORDER BY s.created_at, s.id) AS position
    FROM students s
    JOIN boxes b ON b.id = s.box_id
    WHERE s.person_id IS NOT NULL
      AND s.source IN ('wellhub', 'totalpass')
      AND btrim(COALESCE(s.external_id, '')) <> ''
)
INSERT INTO person_identity_reviews (
    id, organization_id, unit_id, target_person_id, source_person_id,
    reason, provider, created_at
)
SELECT gen_random_uuid(), organization_id, unit_id, target_person_id, person_id,
       'external_identity_conflict', provider, created_at
FROM ranked
WHERE position > 1 AND person_id <> target_person_id
ON CONFLICT DO NOTHING;

-- An athlete account is linked automatically only when every confirmed
-- student link in the organization already points to one person.
WITH candidates AS (
    SELECT b.organization_id, abm.athlete_account_id,
           min(s.person_id::text)::uuid AS person_id,
           count(DISTINCT s.person_id) AS person_count,
           min(asl.linked_at) AS linked_at
    FROM athlete_student_links asl
    JOIN athlete_box_memberships abm ON abm.id = asl.membership_id
    JOIN boxes b ON b.id = abm.box_id
    JOIN students s ON s.id = asl.student_id
    WHERE s.person_id IS NOT NULL
    GROUP BY b.organization_id, abm.athlete_account_id
)
INSERT INTO person_athlete_accounts (
    id, organization_id, person_id, athlete_account_id, link_method, linked_at, created_at
)
SELECT gen_random_uuid(), organization_id, person_id, athlete_account_id,
       'confirmed_student_links', linked_at, linked_at
FROM candidates
WHERE person_count = 1
ON CONFLICT DO NOTHING;

WITH candidates AS (
    SELECT b.organization_id, abm.athlete_account_id,
           min(s.person_id::text)::uuid AS target_person_id,
           max(s.person_id::text)::uuid AS source_person_id,
           min(s.box_id::text)::uuid AS unit_id,
           count(DISTINCT s.person_id) AS person_count,
           min(asl.linked_at) AS linked_at
    FROM athlete_student_links asl
    JOIN athlete_box_memberships abm ON abm.id = asl.membership_id
    JOIN boxes b ON b.id = abm.box_id
    JOIN students s ON s.id = asl.student_id
    WHERE s.person_id IS NOT NULL
    GROUP BY b.organization_id, abm.athlete_account_id
)
INSERT INTO person_identity_reviews (
    id, organization_id, unit_id, target_person_id, source_person_id,
    reason, created_at
)
SELECT gen_random_uuid(), organization_id, unit_id, target_person_id,
       source_person_id, 'athlete_account_conflict', linked_at
FROM candidates
WHERE person_count > 1 AND target_person_id <> source_person_id
ON CONFLICT DO NOTHING;

CREATE TRIGGER set_person_external_identities_updated_at
BEFORE UPDATE ON person_external_identities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
