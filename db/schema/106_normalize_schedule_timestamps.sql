UPDATE job_schedules
   SET next_run_at = datetime('now'),
       last_run_at = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE cast(next_run_at AS TEXT) LIKE '% m=+%'
    OR cast(next_run_at AS TEXT) LIKE '% UTC%'
    OR cast(last_run_at AS TEXT) LIKE '% m=+%'
    OR cast(last_run_at AS TEXT) LIKE '% UTC%';
