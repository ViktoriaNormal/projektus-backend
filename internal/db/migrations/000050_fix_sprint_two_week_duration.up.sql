-- =============================================================================
-- Migration 050: двухнедельные спринты MOBAPP (14 календарных дней)
-- В seed 038 end_date на 2 дня короче, чем sprint_duration_weeks = 2.
-- Формула как в StartSprint: end_date = start_date + (weeks * 7 - 1).
-- ref: 18.05.2026
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_050_sprint_backup;

CREATE TABLE migration_050_sprint_backup AS
SELECT s.id, s.end_date, s.updated_at
FROM sprints s
JOIN projects p ON p.id = s.project_id
WHERE p.id = 'c0000000-0000-0000-0001-000000000000'::uuid
  AND (s.end_date - s.start_date + 1) < (COALESCE(p.sprint_duration_weeks, 2) * 7);

UPDATE sprints s
SET
    end_date   = s.start_date + (GREATEST(COALESCE(p.sprint_duration_weeks, 2), 1) * 7 - 1),
    updated_at = CASE
        WHEN s.status = 'completed' THEN
            ((s.start_date + (GREATEST(COALESCE(p.sprint_duration_weeks, 2), 1) * 7 - 1))::timestamp
                AT TIME ZONE 'Europe/Moscow') + interval '18 hours'
        ELSE s.updated_at
    END
FROM projects p
WHERE s.project_id = p.id
  AND p.id = 'c0000000-0000-0000-0001-000000000000'::uuid
  AND (s.end_date - s.start_date + 1) < (COALESCE(p.sprint_duration_weeks, 2) * 7);
