-- Откат 045: удалить пересобранную историю, восстановить бэкап

DELETE FROM task_status_history
WHERE id >= 'f4500000-0000-0000-0002-000000000001'::uuid
  AND id <= 'f4500000-0000-0000-0003-000000000999'::uuid;

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT id, task_id, column_id, entered_at, left_at
FROM migration_045_history_backup
ON CONFLICT (id) DO NOTHING;

DROP TABLE IF EXISTS migration_045_history_backup;
