-- Sprint tasks: linking tasks to sprints

-- name: AddTaskToSprint :one
INSERT INTO sprint_tasks (sprint_id, task_id, sort_order)
VALUES ($1, $2, $3)
ON CONFLICT (sprint_id, task_id) DO UPDATE
SET sort_order = EXCLUDED.sort_order
RETURNING sprint_id, task_id, sort_order;

-- name: RemoveTaskFromSprint :exec
DELETE FROM sprint_tasks
WHERE sprint_id = $1 AND task_id = $2;

-- name: RemoveTaskFromAllSprints :exec
DELETE FROM sprint_tasks
WHERE task_id = $1;

-- name: GetSprintTasks :many
SELECT st.sprint_id, st.task_id, st.sort_order
FROM sprint_tasks st
JOIN tasks t ON t.id = st.task_id
WHERE st.sprint_id = $1 AND t.deleted_at IS NULL
ORDER BY st.sort_order NULLS LAST;

-- name: UpdateTaskOrder :exec
UPDATE sprint_tasks
SET sort_order = $3
WHERE sprint_id = $1 AND task_id = $2;

-- name: ListSprintTasksFull :many
SELECT t.id, t.key, t.project_id, t.owner_id, t.executor_id, t.name, t.description,
       t.deadline, t.column_id, t.swimlane_id, t.deleted_at, t.created_at,
       t.priority, t.estimation, t.board_id,
       c.name AS column_name, c.system_type AS column_system_type,
       m_owner.user_id AS owner_user_id,
       m_exec.user_id AS executor_user_id
FROM tasks t
JOIN sprint_tasks st ON st.task_id = t.id
LEFT JOIN columns c ON t.column_id = c.id
JOIN members m_owner ON m_owner.id = t.owner_id
LEFT JOIN members m_exec ON m_exec.id = t.executor_id
WHERE st.sprint_id = $1 AND t.deleted_at IS NULL
ORDER BY st.sort_order NULLS LAST;
