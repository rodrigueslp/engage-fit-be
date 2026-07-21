ALTER TABLE automation_runs
    ADD COLUMN schedule_id UUID REFERENCES automation_schedules(id) ON DELETE SET NULL,
    ADD COLUMN execution_key VARCHAR(200),
    ADD COLUMN scheduled_for TIMESTAMPTZ;

CREATE UNIQUE INDEX idx_automation_runs_box_execution_key
    ON automation_runs(box_id, execution_key)
    WHERE execution_key IS NOT NULL AND execution_key <> '';

CREATE INDEX idx_automation_runs_schedule_started
    ON automation_runs(schedule_id, started_at DESC)
    WHERE schedule_id IS NOT NULL;

CREATE INDEX idx_automation_runs_running_started
    ON automation_runs(started_at)
    WHERE status = 'running';
