ALTER TABLE students
    ADD COLUMN membership_started_at DATE,
    ADD COLUMN membership_started_source VARCHAR(32);

UPDATE students s
SET membership_started_at = first_presence.first_checkin,
    membership_started_source = 'first_checkin_inferred'
FROM (
    SELECT student_id, MIN(checkin_date)::date AS first_checkin
    FROM checkins
    GROUP BY student_id
) first_presence
WHERE first_presence.student_id = s.id
  AND s.anonymized_at IS NULL;

ALTER TABLE students
    ADD CONSTRAINT students_membership_started_source_check CHECK (
        membership_started_source IS NULL OR membership_started_source IN (
            'manual',
            'integration',
            'first_checkin_inferred'
        )
    );

CREATE INDEX idx_students_box_membership_started
    ON students (box_id, membership_started_at)
    WHERE anonymized_at IS NULL;
