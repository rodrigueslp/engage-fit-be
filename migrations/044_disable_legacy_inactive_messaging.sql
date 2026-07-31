UPDATE automation_schedules
SET enabled = FALSE,
    updated_at = NOW()
WHERE mode = 'send_inactive'
  AND enabled = TRUE;
