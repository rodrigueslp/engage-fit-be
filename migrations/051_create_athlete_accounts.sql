CREATE TABLE athlete_accounts (
    id UUID PRIMARY KEY,
    name VARCHAR(160) NOT NULL,
    email VARCHAR(320) NOT NULL,
    password_hash TEXT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_accounts_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX idx_athlete_accounts_email_unique
    ON athlete_accounts(LOWER(email));

CREATE TABLE athlete_box_memberships (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_box_memberships_status_check CHECK (status IN ('active', 'inactive')),
    CONSTRAINT athlete_box_memberships_unique UNIQUE (athlete_account_id, box_id)
);

CREATE INDEX idx_athlete_box_memberships_box_status
    ON athlete_box_memberships(box_id, status);

CREATE TABLE athlete_student_links (
    id UUID PRIMARY KEY,
    membership_id UUID NOT NULL REFERENCES athlete_box_memberships(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    link_method VARCHAR(40) NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_student_links_method_check CHECK (link_method IN ('individual_invite', 'assisted_code', 'contact_activation', 'manual_review')),
    CONSTRAINT athlete_student_links_student_unique UNIQUE (student_id),
    CONSTRAINT athlete_student_links_membership_student_unique UNIQUE (membership_id, student_id)
);

CREATE TABLE athlete_invitations (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_invitations_token_unique UNIQUE (token_hash)
);

CREATE INDEX idx_athlete_invitations_student_created
    ON athlete_invitations(student_id, created_at DESC);

CREATE TABLE athlete_sessions (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_sessions_token_unique UNIQUE (token_hash)
);

CREATE INDEX idx_athlete_sessions_account_active
    ON athlete_sessions(athlete_account_id, expires_at DESC)
    WHERE revoked_at IS NULL;
