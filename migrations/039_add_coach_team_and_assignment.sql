ALTER TABLE users
    ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (
        (role IN ('OWNER', 'COACH') AND box_id IS NOT NULL) OR
        (role = 'PLATFORM_ADMIN' AND box_id IS NULL)
    );

CREATE INDEX idx_users_box_role_active
    ON users (box_id, role, active)
    WHERE box_id IS NOT NULL;

ALTER TABLE retention_interventions
    ADD COLUMN assigned_to_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_retention_interventions_box_assignee_status
    ON retention_interventions (box_id, assigned_to_user_id, status, planned_for);
