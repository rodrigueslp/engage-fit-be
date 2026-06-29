CREATE TABLE IF NOT EXISTS automation_schedules (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    run_time CHAR(5) NOT NULL,
    timezone VARCHAR(80) NOT NULL DEFAULT 'America/Sao_Paulo',
    days_of_week VARCHAR(30) NOT NULL DEFAULT '0,1,2,3,4,5,6',
    allow_resend BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'automation_schedules_mode_check'
    ) THEN
        ALTER TABLE automation_schedules
        ADD CONSTRAINT automation_schedules_mode_check
        CHECK (mode IN ('full_daily', 'recalculate_only', 'send_almost_there', 'send_achieved', 'send_inactive'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_automation_schedules_box_id ON automation_schedules(box_id);
CREATE INDEX IF NOT EXISTS idx_automation_schedules_enabled ON automation_schedules(enabled);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'set_automation_schedules_updated_at'
    ) THEN
        CREATE TRIGGER set_automation_schedules_updated_at
        BEFORE UPDATE ON automation_schedules
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;
