ALTER TABLE message_templates
    ADD COLUMN IF NOT EXISTS template_type VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS provider VARCHAR(32) NOT NULL DEFAULT 'twilio',
    ADD COLUMN IF NOT EXISTS approval_status VARCHAR(32) NOT NULL DEFAULT 'NOT_CONFIGURED',
    ADD COLUMN IF NOT EXISTS language VARCHAR(16) NOT NULL DEFAULT 'pt_BR';

CREATE UNIQUE INDEX IF NOT EXISTS idx_message_templates_box_type
    ON message_templates(box_id, template_type)
    WHERE template_type <> '';

ALTER TABLE message_campaigns
    ADD COLUMN IF NOT EXISTS template_type VARCHAR(32) NOT NULL DEFAULT '';

UPDATE message_campaigns
SET template_type = CASE
    WHEN audience IN ('almost_there', 'near_goal') THEN 'ALMOST_THERE'
    WHEN audience = 'achieved' THEN 'GOAL_REACHED'
    WHEN audience = 'inactive' THEN 'WE_MISS_YOU'
    ELSE template_type
END
WHERE template_type = '';
