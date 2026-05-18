-- Откат 047

DELETE FROM task_status_history h
USING tasks t
WHERE h.task_id = t.id
  AND t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  );

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT id, task_id, column_id, entered_at, left_at
FROM migration_047_history_backup
ON CONFLICT (id) DO NOTHING;

DROP TABLE IF EXISTS migration_047_history_backup;
