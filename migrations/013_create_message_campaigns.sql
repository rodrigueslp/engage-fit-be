CREATE TABLE message_campaigns (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    audience VARCHAR(32) NOT NULL,
    template_id UUID NOT NULL REFERENCES message_templates(id) ON DELETE RESTRICT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT message_campaigns_audience_check CHECK (audience IN ('near_goal', 'almost_there', 'achieved', 'inactive', 'all'))
);

CREATE INDEX idx_message_campaigns_box_id ON message_campaigns(box_id);
CREATE INDEX idx_message_campaigns_audience ON message_campaigns(audience);
