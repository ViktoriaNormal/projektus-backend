-- Откат 050: восстановить end_date и updated_at спринтов

UPDATE sprints s
SET
    end_date   = b.end_date,
    updated_at = b.updated_at
FROM migration_050_sprint_backup b
WHERE s.id = b.id;

DROP TABLE IF EXISTS migration_050_sprint_backup;
