-- Sprint tasks: linking tasks to sprints

-- name: AddTaskToSprint :one
INSERT INTO sprint_tasks (sprint_id, task_id, "order")
VALUES ($1, $2, $3)
ON CONFLICT (sprint_id, task_id) DO UPDATE
SET "order" = EXCLUDED."order",
    added_at = NOW()
RETURNING *;

-- name: RemoveTaskFromSprint :exec
DELETE FROM sprint_tasks
WHERE sprint_id = $1 AND task_id = $2;

-- name: GetSprintTasks :many
SELECT *
FROM sprint_tasks
WHERE sprint_id = $1
ORDER BY "order" NULLS LAST, added_at;

-- name: UpdateTaskOrder :exec
UPDATE sprint_tasks
SET "order" = $3
WHERE sprint_id = $1 AND task_id = $2;

