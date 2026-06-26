CREATE TABLE campaign_progresses (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    current_checkins INTEGER NOT NULL DEFAULT 0,
    target_checkins INTEGER NOT NULL,
    progress_percentage NUMERIC(6, 2) NOT NULL DEFAULT 0,
    achieved BOOLEAN NOT NULL DEFAULT FALSE,
    near_goal BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT campaign_progresses_current_checkins_check CHECK (current_checkins >= 0),
    CONSTRAINT campaign_progresses_target_checkins_check CHECK (target_checkins > 0),
    CONSTRAINT campaign_progresses_percentage_check CHECK (progress_percentage >= 0),
    CONSTRAINT campaign_progresses_campaign_student_unique UNIQUE (campaign_id, student_id)
);

CREATE INDEX idx_campaign_progresses_campaign_id ON campaign_progresses(campaign_id);
CREATE INDEX idx_campaign_progresses_student_id ON campaign_progresses(student_id);
CREATE INDEX idx_campaign_progresses_campaign_near_goal ON campaign_progresses(campaign_id, near_goal);
CREATE INDEX idx_campaign_progresses_campaign_achieved ON campaign_progresses(campaign_id, achieved);
