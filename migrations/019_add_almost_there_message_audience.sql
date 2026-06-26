ALTER TABLE message_campaigns
DROP CONSTRAINT IF EXISTS message_campaigns_audience_check;

ALTER TABLE message_campaigns
ADD CONSTRAINT message_campaigns_audience_check
CHECK (audience IN ('near_goal', 'almost_there', 'achieved', 'inactive', 'all'));
