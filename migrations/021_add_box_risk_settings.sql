ALTER TABLE boxes
ADD COLUMN IF NOT EXISTS risk_inactive_days INTEGER NOT NULL DEFAULT 7,
ADD COLUMN IF NOT EXISTS risk_message_cooldown_days INTEGER NOT NULL DEFAULT 14;

ALTER TABLE boxes
DROP CONSTRAINT IF EXISTS boxes_risk_inactive_days_check;

ALTER TABLE boxes
ADD CONSTRAINT boxes_risk_inactive_days_check
CHECK (risk_inactive_days >= 1 AND risk_inactive_days <= 365);

ALTER TABLE boxes
DROP CONSTRAINT IF EXISTS boxes_risk_message_cooldown_days_check;

ALTER TABLE boxes
ADD CONSTRAINT boxes_risk_message_cooldown_days_check
CHECK (risk_message_cooldown_days >= 1 AND risk_message_cooldown_days <= 365);
