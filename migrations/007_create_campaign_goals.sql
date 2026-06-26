CREATE TABLE campaign_goals (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    source VARCHAR(32) NOT NULL,
    target_checkins INTEGER NOT NULL,

    CONSTRAINT campaign_goals_source_check CHECK (source IN ('wellhub', 'totalpass')),
    CONSTRAINT campaign_goals_target_check CHECK (target_checkins > 0),
    CONSTRAINT campaign_goals_campaign_source_unique UNIQUE (campaign_id, source)
);
