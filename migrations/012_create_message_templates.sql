CREATE TABLE message_templates (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_message_templates_box_id ON message_templates(box_id);
