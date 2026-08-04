ALTER TABLE import_histories
    ADD COLUMN status VARCHAR(24) NOT NULL DEFAULT 'completed',
    ADD COLUMN students_created INTEGER,
    ADD COLUMN checkins_created INTEGER,
    ADD COLUMN completed_at TIMESTAMPTZ,
    ADD COLUMN error_code VARCHAR(64);

ALTER TABLE import_histories
    ADD CONSTRAINT import_histories_status_check
        CHECK (status IN ('processing', 'completed', 'failed')),
    ADD CONSTRAINT import_histories_students_created_check
        CHECK (students_created IS NULL OR students_created >= 0),
    ADD CONSTRAINT import_histories_checkins_created_check
        CHECK (checkins_created IS NULL OR checkins_created >= 0);

UPDATE import_histories AS history
SET checkins_created = totals.total,
    completed_at = history.imported_at
FROM (
    SELECT import_history_id, COUNT(*)::INTEGER AS total
    FROM checkins
    WHERE import_history_id IS NOT NULL
    GROUP BY import_history_id
) AS totals
WHERE totals.import_history_id = history.id;

UPDATE import_histories
SET checkins_created = 0,
    completed_at = imported_at
WHERE checkins_created IS NULL;

CREATE INDEX idx_import_histories_box_status_imported
    ON import_histories(box_id, status, imported_at DESC);
