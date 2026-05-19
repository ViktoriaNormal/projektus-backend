-- Откат 053: удалить f053*, восстановить бэкап истории и колонок задач

DELETE FROM task_status_history
WHERE id >= 'f0530000-0000-0000-0002-000000000001'::uuid
  AND id <= 'f0530000-0000-0000-0003-000000000999'::uuid;

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT id, task_id, column_id, entered_at, left_at
FROM migration_053_history_backup
ON CONFLICT (id) DO NOTHING;

UPDATE tasks t
SET
    column_id = b.column_id,
    deadline  = b.deadline
FROM migration_053_tasks_backup b
WHERE t.id = b.id;

DROP TABLE IF EXISTS migration_053_history_backup;
DROP TABLE IF EXISTS migration_053_tasks_backup;
