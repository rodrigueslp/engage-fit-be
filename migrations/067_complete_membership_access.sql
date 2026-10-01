ALTER TABLE students
    ADD COLUMN legacy_access_eligible BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE students s
SET legacy_access_eligible = TRUE
WHERE s.source = 'box_member'
  AND s.anonymized_at IS NULL
  AND s.membership_started_at IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM enrollments e
      WHERE e.person_id = s.person_id AND e.home_unit_id = s.box_id
  );

CREATE INDEX idx_students_legacy_access_transition
    ON students(box_id, person_id)
    WHERE legacy_access_eligible = TRUE AND anonymized_at IS NULL;

ALTER TABLE attendance_access_decisions
    ADD COLUMN entitlement_id UUID,
    ADD COLUMN remaining_usage INTEGER;

ALTER TABLE attendance_access_decisions
    ADD CONSTRAINT attendance_decisions_remaining_usage_check
    CHECK (remaining_usage IS NULL OR remaining_usage >= 0);
