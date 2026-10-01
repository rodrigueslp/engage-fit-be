CREATE TABLE announcements (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    title VARCHAR(120) NOT NULL,
    body TEXT NOT NULL,
    priority VARCHAR(16) NOT NULL DEFAULT 'info',
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    audience_kind VARCHAR(16) NOT NULL,
    audience_value VARCHAR(255),
    publish_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id, unit_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT announcements_title_check CHECK (char_length(btrim(title)) BETWEEN 3 AND 120),
    CONSTRAINT announcements_body_check CHECK (char_length(btrim(body)) BETWEEN 3 AND 4000),
    CONSTRAINT announcements_priority_check CHECK (priority IN ('info','important','urgent')),
    CONSTRAINT announcements_status_check CHECK (status IN ('draft','scheduled','published','archived')),
    CONSTRAINT announcements_audience_check CHECK (
        (audience_kind = 'all' AND audience_value IS NULL) OR
        (audience_kind = 'source' AND audience_value IN ('box_member','wellhub','totalpass')) OR
        (audience_kind = 'plan' AND audience_value IS NOT NULL)
    ),
    CONSTRAINT announcements_publication_check CHECK (
        (status = 'draft' AND publish_at IS NULL AND published_at IS NULL AND archived_at IS NULL) OR
        (status = 'scheduled' AND publish_at IS NOT NULL AND published_at IS NULL AND archived_at IS NULL) OR
        (status = 'published' AND publish_at IS NOT NULL AND published_at IS NOT NULL AND archived_at IS NULL) OR
        (status = 'archived' AND publish_at IS NOT NULL AND archived_at IS NOT NULL)
    )
);
CREATE INDEX idx_announcements_unit_status_publish ON announcements(unit_id, status, publish_at DESC);
CREATE TRIGGER set_announcements_updated_at BEFORE UPDATE ON announcements FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE announcement_recipients (
    id UUID PRIMARY KEY,
    announcement_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    athlete_membership_id UUID NOT NULL REFERENCES athlete_box_memberships(id) ON DELETE CASCADE,
    person_id UUID NOT NULL,
    first_viewed_at TIMESTAMPTZ,
    last_viewed_at TIMESTAMPTZ,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (announcement_id, organization_id, unit_id) REFERENCES announcements(id, organization_id, unit_id) ON DELETE CASCADE,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT announcement_recipients_unique UNIQUE (announcement_id, athlete_account_id, athlete_membership_id),
    CONSTRAINT announcement_recipients_view_check CHECK (
        (view_count = 0 AND first_viewed_at IS NULL AND last_viewed_at IS NULL) OR
        (view_count > 0 AND first_viewed_at IS NOT NULL AND last_viewed_at IS NOT NULL AND last_viewed_at >= first_viewed_at)
    )
);
CREATE INDEX idx_announcement_recipients_athlete ON announcement_recipients(athlete_account_id, created_at DESC);
CREATE INDEX idx_announcement_recipients_announcement_viewed ON announcement_recipients(announcement_id, first_viewed_at);
