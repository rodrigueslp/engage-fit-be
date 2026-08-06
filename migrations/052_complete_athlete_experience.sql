ALTER TABLE athlete_accounts
    ADD COLUMN email_verified_at TIMESTAMPTZ;

CREATE TABLE athlete_account_tokens (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    purpose VARCHAR(32) NOT NULL,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_account_tokens_purpose_check CHECK (purpose IN ('verify_email', 'reset_password'))
);

CREATE INDEX idx_athlete_account_tokens_active
    ON athlete_account_tokens(athlete_account_id, purpose, expires_at DESC)
    WHERE used_at IS NULL;

CREATE TABLE athlete_workout_results (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    membership_id UUID NOT NULL REFERENCES athlete_box_memberships(id) ON DELETE CASCADE,
    scale VARCHAR(24) NOT NULL DEFAULT 'scaled',
    entries JSONB NOT NULL DEFAULT '[]'::jsonb,
    rpe SMALLINT,
    notes TEXT NOT NULL DEFAULT '',
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_workout_results_scale_check CHECK (scale IN ('rx', 'scaled', 'adapted')),
    CONSTRAINT athlete_workout_results_rpe_check CHECK (rpe IS NULL OR rpe BETWEEN 1 AND 10),
    CONSTRAINT athlete_workout_results_entries_array CHECK (jsonb_typeof(entries) = 'array'),
    CONSTRAINT athlete_workout_results_unique UNIQUE (athlete_account_id, workout_id)
);

CREATE INDEX idx_athlete_workout_results_history
    ON athlete_workout_results(athlete_account_id, performed_at DESC);

CREATE TABLE athlete_personal_records (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    movement_key VARCHAR(160) NOT NULL,
    movement_name VARCHAR(160) NOT NULL,
    metric VARCHAR(24) NOT NULL,
    best_value NUMERIC(12, 3) NOT NULL,
    unit VARCHAR(16) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'estimated',
    source_result_id UUID NOT NULL REFERENCES athlete_workout_results(id) ON DELETE CASCADE,
    achieved_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_personal_records_metric_check CHECK (metric IN ('load', 'reps', 'time')),
    CONSTRAINT athlete_personal_records_status_check CHECK (status IN ('estimated', 'confirmed')),
    CONSTRAINT athlete_personal_records_value_check CHECK (best_value > 0),
    CONSTRAINT athlete_personal_records_unique UNIQUE (athlete_account_id, movement_key, metric)
);

CREATE INDEX idx_athlete_personal_records_account
    ON athlete_personal_records(athlete_account_id, movement_name);

CREATE TABLE athlete_workout_insights (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    input_hash CHAR(64) NOT NULL,
    provider VARCHAR(40) NOT NULL,
    model VARCHAR(100) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_workout_insights_unique UNIQUE (athlete_account_id, workout_id, input_hash)
);

