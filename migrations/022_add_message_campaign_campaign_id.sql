ALTER TABLE message_campaigns
    ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL;

UPDATE message_campaigns
SET campaign_id = campaigns.id
FROM campaigns
WHERE message_campaigns.campaign_id IS NULL
  AND campaigns.box_id = message_campaigns.box_id
  AND campaigns.id = (
      SELECT c.id
      FROM campaigns c
      WHERE c.box_id = message_campaigns.box_id
      ORDER BY c.active DESC, c.end_date DESC, c.created_at DESC
      LIMIT 1
  );

CREATE INDEX IF NOT EXISTS idx_message_campaigns_campaign_id ON message_campaigns(campaign_id);
