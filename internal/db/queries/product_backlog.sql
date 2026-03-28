-- Backlog ordering

-- name: AddToProductBacklog :one
INSERT INTO backlog (project_id, task_id, sort_order)
VALUES ($1, $2, $3)
RETURNING project_id, task_id, sort_order;

-- name: RemoveFromProductBacklog :exec
DELETE FROM backlog
WHERE project_id = $1 AND task_id = $2;

-- name: GetProductBacklog :many
SELECT project_id, task_id, sort_order
FROM backlog
WHERE project_id = $1
ORDER BY sort_order;

-- name: UpdateProductBacklogOrder :exec
UPDATE backlog
SET sort_order = $3
WHERE project_id = $1 AND task_id = $2;
