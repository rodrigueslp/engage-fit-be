CREATE TABLE message_recipients (
    id UUID PRIMARY KEY,
    message_campaign_id UUID NOT NULL REFERENCES message_campaigns(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    phone VARCHAR(50) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT message_recipients_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX idx_message_recipients_campaign_id ON message_recipients(message_campaign_id);
CREATE INDEX idx_message_recipients_student_id ON message_recipients(student_id);
CREATE INDEX idx_message_recipients_status ON message_recipients(status);
