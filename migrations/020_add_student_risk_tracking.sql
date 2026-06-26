ALTER TABLE students
ADD COLUMN IF NOT EXISTS risk_status VARCHAR(32) NOT NULL DEFAULT 'active',
ADD COLUMN IF NOT EXISTS risk_last_message_at TIMESTAMPTZ;

ALTER TABLE students
DROP CONSTRAINT IF EXISTS students_risk_status_check;

ALTER TABLE students
ADD CONSTRAINT students_risk_status_check
CHECK (risk_status IN ('active', 'observing', 'paused', 'not_interested'));

CREATE INDEX IF NOT EXISTS idx_students_box_risk_status ON students(box_id, risk_status);
