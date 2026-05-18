-- Откат 040: убрать привязку багов баг-трекера MOBAPP к спринтам.

DELETE FROM sprint_tasks st
USING tasks t, boards b, projects p
WHERE st.task_id = t.id
  AND t.board_id = b.id
  AND b.project_id = p.id
  AND p.key = 'MOBAPP'
  AND b.is_default = false;
