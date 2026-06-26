CREATE TABLE campaigns (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT campaigns_date_check CHECK (end_date >= start_date)
);

CREATE INDEX idx_campaigns_box_id ON campaigns(box_id);
CREATE INDEX idx_campaigns_box_active ON campaigns(box_id, active);
