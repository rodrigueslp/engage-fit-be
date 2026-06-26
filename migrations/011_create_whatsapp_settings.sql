CREATE TABLE whatsapp_settings (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL UNIQUE REFERENCES boxes(id) ON DELETE CASCADE,
    base_url VARCHAR(500) NOT NULL,
    instance_name VARCHAR(255) NOT NULL,
    api_key_encrypted TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
