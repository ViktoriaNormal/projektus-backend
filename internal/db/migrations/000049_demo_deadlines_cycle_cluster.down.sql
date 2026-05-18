-- Откат 049

DELETE FROM task_dependencies
WHERE id >= 'f0490000-0000-0000-0001-000000000001'::uuid
  AND id <= 'f0490000-0000-0000-0002-000000000099'::uuid;

UPDATE tasks t
SET deadline = b.deadline
FROM migration_049_deadline_backup b
WHERE t.id = b.task_id;

UPDATE task_status_history h
SET entered_at = b.entered_at,
    left_at    = b.left_at
FROM migration_049_history_backup b
WHERE h.id = b.id;

DROP TABLE IF EXISTS migration_049_deadline_backup;
DROP TABLE IF EXISTS migration_049_history_backup;
