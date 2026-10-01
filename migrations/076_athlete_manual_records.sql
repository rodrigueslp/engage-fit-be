CREATE TABLE athlete_manual_records (
    id UUID PRIMARY KEY,
    athlete_account_id UUID NOT NULL REFERENCES athlete_accounts(id) ON DELETE CASCADE,
    membership_id UUID NOT NULL REFERENCES athlete_box_memberships(id) ON DELETE CASCADE,
    movement_key VARCHAR(160) NOT NULL,
    movement_name VARCHAR(160) NOT NULL,
    metric VARCHAR(24) NOT NULL,
    best_value NUMERIC(12, 3) NOT NULL,
    unit VARCHAR(16) NOT NULL,
    achieved_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT athlete_manual_records_metric_check CHECK (metric IN ('load', 'reps', 'time')),
    CONSTRAINT athlete_manual_records_value_check CHECK (best_value > 0)
);

CREATE INDEX idx_athlete_manual_records_account
    ON athlete_manual_records(athlete_account_id, movement_key, metric, achieved_at DESC);
