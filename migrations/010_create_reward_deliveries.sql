CREATE TABLE reward_deliveries (
    id UUID PRIMARY KEY,
    reward_id UUID NOT NULL REFERENCES rewards(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    delivered BOOLEAN NOT NULL DEFAULT FALSE,
    delivered_at TIMESTAMPTZ,

    CONSTRAINT reward_deliveries_reward_student_unique UNIQUE (reward_id, student_id),
    CONSTRAINT reward_deliveries_delivered_at_check CHECK (
        (delivered = FALSE AND delivered_at IS NULL)
        OR (delivered = TRUE AND delivered_at IS NOT NULL)
    )
);

CREATE INDEX idx_reward_deliveries_reward_id ON reward_deliveries(reward_id);
CREATE INDEX idx_reward_deliveries_student_id ON reward_deliveries(student_id);
CREATE INDEX idx_reward_deliveries_delivered ON reward_deliveries(delivered);
