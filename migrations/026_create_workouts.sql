CREATE TABLE IF NOT EXISTS workouts (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    workout_date DATE NOT NULL,
    title VARCHAR(160) NOT NULL,
    goal VARCHAR(80) NOT NULL DEFAULT '',
    movements TEXT NOT NULL DEFAULT '',
    coach_notes TEXT NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT workouts_status_check CHECK (status IN ('draft', 'published'))
);

CREATE INDEX IF NOT EXISTS idx_workouts_box_date ON workouts(box_id, workout_date DESC);

CREATE TABLE IF NOT EXISTS workout_message_drafts (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    audience VARCHAR(32) NOT NULL,
    generated_body TEXT NOT NULL,
    approved_body TEXT NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL DEFAULT 'draft',
    total_recipients INTEGER NOT NULL DEFAULT 0,
    sent_recipients INTEGER NOT NULL DEFAULT 0,
    failed_recipients INTEGER NOT NULL DEFAULT 0,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    CONSTRAINT workout_message_drafts_audience_check CHECK (audience IN ('near_goal', 'almost_there', 'achieved', 'inactive', 'all')),
    CONSTRAINT workout_message_drafts_status_check CHECK (status IN ('draft', 'approved', 'sent'))
);

CREATE INDEX IF NOT EXISTS idx_workout_message_drafts_workout_id ON workout_message_drafts(workout_id);
CREATE INDEX IF NOT EXISTS idx_workout_message_drafts_box_id ON workout_message_drafts(box_id);

CREATE TABLE IF NOT EXISTS workout_message_recipients (
    id UUID PRIMARY KEY,
    workout_message_draft_id UUID NOT NULL REFERENCES workout_message_drafts(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    phone VARCHAR(40) NOT NULL,
    status VARCHAR(24) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT workout_message_recipients_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_workout_message_recipients_draft_id ON workout_message_recipients(workout_message_draft_id);

CREATE TABLE IF NOT EXISTS llm_generation_logs (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    draft_id UUID REFERENCES workout_message_drafts(id) ON DELETE SET NULL,
    provider VARCHAR(40) NOT NULL,
    model VARCHAR(80) NOT NULL,
    prompt_summary TEXT NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_generation_logs_workout_id ON llm_generation_logs(workout_id);
