CREATE UNIQUE INDEX IF NOT EXISTS idx_checkins_unique_visit
ON checkins (box_id, source, student_id, checkin_date, checkin_time);
