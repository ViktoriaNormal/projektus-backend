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

-- name: GetSprintTasks :many
SELECT sprint_id, task_id, sort_order
FROM sprint_tasks
WHERE sprint_id = $1
ORDER BY sort_order NULLS LAST;

-- name: UpdateTaskOrder :exec
UPDATE sprint_tasks
SET sort_order = $3
WHERE sprint_id = $1 AND task_id = $2;
