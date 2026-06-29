CREATE TABLE email_settings (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL UNIQUE REFERENCES boxes(id) ON DELETE CASCADE,
    provider VARCHAR(30) NOT NULL DEFAULT 'smtp',
    smtp_host VARCHAR(255) NOT NULL DEFAULT '',
    smtp_port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255) NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    from_email VARCHAR(255) NOT NULL DEFAULT '',
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_settings_provider_check CHECK (provider IN ('smtp', 'mock'))
);

CREATE TABLE email_templates (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE email_campaigns (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    audience VARCHAR(50) NOT NULL,
    template_id UUID NOT NULL REFERENCES email_templates(id) ON DELETE RESTRICT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_campaigns_audience_check CHECK (audience IN ('near_goal', 'almost_there', 'achieved', 'inactive', 'all'))
);

CREATE TABLE email_recipients (
    id UUID PRIMARY KEY,
    email_campaign_id UUID NOT NULL REFERENCES email_campaigns(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_recipients_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE TABLE automation_runs (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT '',
    filename VARCHAR(255) NOT NULL DEFAULT '',
    imported BOOLEAN NOT NULL DEFAULT false,
    recalculated_campaigns INTEGER NOT NULL DEFAULT 0,
    skipped_message_campaigns INTEGER NOT NULL DEFAULT 0,
    sent_messages INTEGER NOT NULL DEFAULT 0,
    failed_messages INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    CONSTRAINT automation_runs_status_check CHECK (status IN ('running', 'success', 'failed'))
);

CREATE INDEX idx_email_templates_box_id ON email_templates(box_id);
CREATE INDEX idx_email_campaigns_box_id ON email_campaigns(box_id);
CREATE INDEX idx_email_campaigns_campaign_id ON email_campaigns(campaign_id);
CREATE INDEX idx_email_recipients_campaign_id ON email_recipients(email_campaign_id);
CREATE INDEX idx_email_recipients_status ON email_recipients(status);
CREATE INDEX idx_automation_runs_box_id ON automation_runs(box_id);
CREATE INDEX idx_automation_runs_started_at ON automation_runs(started_at DESC);

CREATE TRIGGER set_email_settings_updated_at
BEFORE UPDATE ON email_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_email_templates_updated_at
BEFORE UPDATE ON email_templates
FOR EACH ROW EXECUTE FUNCTION set_updated_at();


CREATE TABLE automation_schedules (
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT automation_schedules_mode_check CHECK (mode IN ('full_daily', 'recalculate_only', 'send_almost_there', 'send_achieved', 'send_inactive'))
);

CREATE INDEX idx_automation_schedules_box_id ON automation_schedules(box_id);
CREATE INDEX idx_automation_schedules_enabled ON automation_schedules(enabled);

CREATE TRIGGER set_automation_schedules_updated_at
BEFORE UPDATE ON automation_schedules
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
