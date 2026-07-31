ALTER TABLE boxes
ADD COLUMN IF NOT EXISTS retention_baseline_at DATE;

UPDATE boxes
SET retention_baseline_at = CURRENT_DATE
WHERE retention_baseline_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM import_histories i
    WHERE i.box_id = boxes.id
  );

ALTER TABLE students
ADD COLUMN IF NOT EXISTS retention_monitoring_status VARCHAR(20) NOT NULL DEFAULT 'monitored',
ADD COLUMN IF NOT EXISTS retention_exclusion_reason VARCHAR(40),
ADD COLUMN IF NOT EXISTS retention_excluded_until DATE,
ADD COLUMN IF NOT EXISTS retention_excluded_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS retention_excluded_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE students
DROP CONSTRAINT IF EXISTS students_retention_monitoring_status_check;

ALTER TABLE students
ADD CONSTRAINT students_retention_monitoring_status_check
CHECK (retention_monitoring_status IN ('monitored', 'excluded'));

CREATE INDEX IF NOT EXISTS idx_students_retention_monitoring
ON students (box_id, retention_monitoring_status, retention_excluded_until)
WHERE anonymized_at IS NULL;

CREATE TABLE IF NOT EXISTS retention_monitoring_events (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    monitoring_status VARCHAR(20) NOT NULL,
    reason VARCHAR(40),
    excluded_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT retention_monitoring_events_status_check
        CHECK (monitoring_status IN ('monitored', 'excluded'))
);

CREATE INDEX IF NOT EXISTS idx_retention_monitoring_events_student
ON retention_monitoring_events (box_id, student_id, created_at DESC);
