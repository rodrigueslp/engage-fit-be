CREATE TABLE rewards (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT rewards_quantity_check CHECK (quantity >= 0)
);

CREATE INDEX idx_rewards_campaign_id ON rewards(campaign_id);
