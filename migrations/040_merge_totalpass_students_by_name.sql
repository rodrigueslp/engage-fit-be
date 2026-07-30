-- TotalPass' token export has no stable collaborator ID. Its ID and Código
-- columns identify each token/check-in, which caused one student per visit.
-- Consolidate those records by normalized collaborator name within each box.

CREATE TEMP TABLE totalpass_student_merge ON COMMIT DROP AS
WITH normalized AS (
    SELECT
        id AS old_id,
        box_id,
        LOWER(REGEXP_REPLACE(BTRIM(name), '[[:space:]]+', ' ', 'g')) AS identity,
        created_at
    FROM students
    WHERE source = 'totalpass'
      AND anonymized_at IS NULL
),
ranked AS (
    SELECT
        old_id,
        identity,
        FIRST_VALUE(old_id) OVER (
            PARTITION BY box_id, identity
            ORDER BY created_at, old_id
        ) AS canonical_id
    FROM normalized
)
SELECT old_id, canonical_id, identity
FROM ranked;

CREATE UNIQUE INDEX totalpass_student_merge_old_id
    ON totalpass_student_merge (old_id);

-- Remove repeated imports of the same visit before all check-ins begin to
-- reference the same canonical student.
DELETE FROM checkins
WHERE id IN (
    SELECT id
    FROM (
        SELECT
            c.id,
            ROW_NUMBER() OVER (
                PARTITION BY c.box_id, c.source, m.canonical_id, c.checkin_date, c.checkin_time
                ORDER BY c.created_at, c.id
            ) AS occurrence
        FROM checkins c
        JOIN totalpass_student_merge m ON m.old_id = c.student_id
        WHERE c.source = 'totalpass'
    ) ranked_checkins
    WHERE occurrence > 1
);

UPDATE checkins c
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE c.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

-- Delivery records carry manual state, so retain the delivered record when a
-- reward has more than one row for students that are now being merged.
DELETE FROM reward_deliveries
WHERE id IN (
    SELECT id
    FROM (
        SELECT
            rd.id,
            ROW_NUMBER() OVER (
                PARTITION BY rd.reward_id, m.canonical_id
                ORDER BY rd.delivered DESC, rd.delivered_at NULLS LAST, rd.id
            ) AS occurrence
        FROM reward_deliveries rd
        JOIN totalpass_student_merge m ON m.old_id = rd.student_id
    ) ranked_deliveries
    WHERE occurrence > 1
);

UPDATE reward_deliveries rd
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE rd.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE message_recipients mr
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE mr.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE email_recipients er
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE er.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE workout_message_recipients wr
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE wr.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE retention_interventions ri
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE ri.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE privacy_audit_events pae
SET student_id = m.canonical_id
FROM totalpass_student_merge m
WHERE pae.student_id = m.old_id
  AND m.old_id <> m.canonical_id;

-- Campaign progress is derived from check-ins. Rebuild it after consolidation
-- so the UI immediately shows the correct totals.
DELETE FROM campaign_progresses cp
USING students s
WHERE cp.student_id = s.id
  AND s.source = 'totalpass'
  AND s.anonymized_at IS NULL;

DELETE FROM students s
USING totalpass_student_merge m
WHERE s.id = m.old_id
  AND m.old_id <> m.canonical_id;

UPDATE students s
SET
    external_id = m.identity,
    membership_started_at = CASE
        WHEN first_checkin.first_date IS NOT NULL
         AND (s.membership_started_at IS NULL OR s.membership_started_source IN ('', 'first_checkin_inferred'))
            THEN first_checkin.first_date
        ELSE s.membership_started_at
    END,
    membership_started_source = CASE
        WHEN first_checkin.first_date IS NOT NULL
         AND (s.membership_started_at IS NULL OR s.membership_started_source IN ('', 'first_checkin_inferred'))
            THEN 'first_checkin_inferred'
        ELSE s.membership_started_source
    END,
    updated_at = NOW()
FROM totalpass_student_merge m
LEFT JOIN LATERAL (
    SELECT MIN(c.checkin_date) AS first_date
    FROM checkins c
    WHERE c.student_id = m.canonical_id
) first_checkin ON TRUE
WHERE s.id = m.canonical_id;

INSERT INTO campaign_progresses (
    id,
    campaign_id,
    student_id,
    current_checkins,
    target_checkins,
    progress_percentage,
    achieved,
    near_goal,
    updated_at
)
SELECT
    GEN_RANDOM_UUID(),
    campaign.id,
    student.id,
    COUNT(checkin.id)::INTEGER,
    goal.target_checkins,
    COUNT(checkin.id) * 100.0 / goal.target_checkins,
    COUNT(checkin.id) >= goal.target_checkins,
    COUNT(checkin.id) * 100.0 / goal.target_checkins >= 80,
    NOW()
FROM campaigns campaign
JOIN campaign_goals goal
  ON goal.campaign_id = campaign.id
 AND goal.source = 'totalpass'
JOIN students student
  ON student.box_id = campaign.box_id
 AND student.source = 'totalpass'
 AND student.anonymized_at IS NULL
JOIN checkins checkin
  ON checkin.student_id = student.id
 AND checkin.checkin_date BETWEEN campaign.start_date AND campaign.end_date
GROUP BY campaign.id, student.id, goal.target_checkins;

-- Match the application's reward synchronization for newly aggregated goals.
DELETE FROM reward_deliveries delivery
USING rewards reward, campaigns campaign, students student
WHERE delivery.reward_id = reward.id
  AND reward.campaign_id = campaign.id
  AND delivery.student_id = student.id
  AND student.source = 'totalpass'
  AND student.anonymized_at IS NULL
  AND delivery.delivered = FALSE
  AND NOT EXISTS (
      SELECT 1
      FROM campaign_progresses progress
      WHERE progress.campaign_id = campaign.id
        AND progress.student_id = student.id
        AND progress.achieved = TRUE
  );

INSERT INTO reward_deliveries (id, reward_id, student_id, delivered, delivered_at)
SELECT
    GEN_RANDOM_UUID(),
    reward.id,
    progress.student_id,
    FALSE,
    NULL
FROM rewards reward
JOIN campaign_progresses progress
  ON progress.campaign_id = reward.campaign_id
 AND progress.achieved = TRUE
JOIN students student
  ON student.id = progress.student_id
 AND student.source = 'totalpass'
 AND student.anonymized_at IS NULL
ON CONFLICT (reward_id, student_id) DO NOTHING;
