ALTER TABLE workouts
    ADD COLUMN raw_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN classification JSONB NOT NULL DEFAULT '{"version":"unclassified","generated_by":"migration","suggested_title":"Treino do dia","sections":[],"formats":[],"movement_mentions":[]}'::jsonb,
    ADD COLUMN classified_at TIMESTAMPTZ;

UPDATE workouts
SET raw_text = movements
WHERE raw_text = '';

ALTER TABLE workouts
    ADD CONSTRAINT workouts_raw_text_size_check
        CHECK (char_length(raw_text) <= 20000);

CREATE INDEX idx_workouts_box_published_date
    ON workouts(box_id, workout_date DESC)
    WHERE status = 'published';
