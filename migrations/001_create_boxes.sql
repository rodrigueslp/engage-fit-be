CREATE TABLE boxes (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    risk_inactive_days INTEGER NOT NULL DEFAULT 7,
    risk_message_cooldown_days INTEGER NOT NULL DEFAULT 14,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT boxes_risk_inactive_days_check CHECK (risk_inactive_days >= 1 AND risk_inactive_days <= 365),
    CONSTRAINT boxes_risk_message_cooldown_days_check CHECK (risk_message_cooldown_days >= 1 AND risk_message_cooldown_days <= 365)
);
