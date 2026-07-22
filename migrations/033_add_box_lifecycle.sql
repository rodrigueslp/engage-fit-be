ALTER TABLE boxes
    ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'active',
    ADD COLUMN status_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN status_changed_at TIMESTAMPTZ,
    ADD COLUMN status_changed_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE boxes
    ADD CONSTRAINT boxes_status_check CHECK (status IN ('active', 'suspended', 'archived'));

CREATE INDEX idx_boxes_status ON boxes(status);
