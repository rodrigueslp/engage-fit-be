ALTER TABLE boxes
    ADD COLUMN access_control_mode VARCHAR(16) NOT NULL DEFAULT 'disabled';

ALTER TABLE boxes
    ADD CONSTRAINT boxes_access_control_mode_check
    CHECK (access_control_mode IN ('disabled', 'manual', 'integrated'));
