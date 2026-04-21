-- Migration 000036: Удаляет из системы типы полей `sprint` и `sprint_list`.
-- Привязка задачи к спринту обеспечивается first-class-механизмом (sprint_tasks
-- + /sprints/{id}/tasks, /projects/{id}/backlog/move-to-sprint), а кастомное
-- поле типа «спринт» параллельно с ним только добавляло спец-кейсов в
-- валидации и резолверы значений. Миграция деструктивная: down-шаг не
-- восстанавливает удалённые строки.
--
-- Перед применением имеет смысл снять счётчики, сколько записей затронет:
--   SELECT field_type, COUNT(*) FROM project_params
--     WHERE field_type IN ('sprint', 'sprint_list') GROUP BY field_type;
--   SELECT field_type, COUNT(*) FROM board_fields
--     WHERE field_type IN ('sprint', 'sprint_list') GROUP BY field_type;
--   SELECT COUNT(*) FROM task_field_values tfv
--     JOIN board_fields bf ON bf.id = tfv.field_id
--     WHERE bf.field_type IN ('sprint', 'sprint_list');

-- Параметры проектов/шаблонов: по openapi они и так запрещены, но страхуемся.
DELETE FROM project_params
WHERE field_type IN ('sprint', 'sprint_list');

-- Кастомные поля доски. task_field_values.field_id имеет ON DELETE CASCADE
-- на board_fields(id), поэтому значения задач по этим полям подчищаются
-- автоматически.
DELETE FROM board_fields
WHERE field_type IN ('sprint', 'sprint_list');
