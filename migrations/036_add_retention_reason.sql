ALTER TABLE retention_interventions
    ADD COLUMN reason_code VARCHAR(32);

ALTER TABLE retention_interventions
    ADD CONSTRAINT retention_interventions_reason_check CHECK (
        reason_code IS NULL OR reason_code IN (
            'travel',
            'schedule',
            'financial',
            'motivation',
            'service',
            'health',
            'moved',
            'unknown',
            'other'
        )
    );

CREATE INDEX idx_retention_interventions_box_reason_completed
    ON retention_interventions (box_id, reason_code, completed_at)
    WHERE status = 'completed';
