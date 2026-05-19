-- Откат 054: удалить f054* и новые задачи, восстановить бэкап

DELETE FROM task_status_history
WHERE id >= 'f0540000-0000-0000-0002-000000000001'::uuid
  AND id <= 'f0540000-0000-0000-0003-000000000999'::uuid;

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT id, task_id, column_id, entered_at, left_at
FROM migration_054_history_backup
ON CONFLICT (id) DO NOTHING;

UPDATE tasks t
SET column_id = b.column_id, deadline = b.deadline
FROM migration_054_tasks_backup b
WHERE t.id = b.id;

DELETE FROM tasks t
USING migration_054_new_tasks n
WHERE t.id = n.id;

DROP TABLE IF EXISTS migration_054_history_backup;
DROP TABLE IF EXISTS migration_054_tasks_backup;
DROP TABLE IF EXISTS migration_054_new_tasks;
