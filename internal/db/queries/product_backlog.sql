-- Backlog ordering

-- name: AddToProductBacklog :one
INSERT INTO backlog (project_id, task_id, sort_order)
VALUES ($1, $2, $3)
RETURNING project_id, task_id, sort_order;

-- name: RemoveFromProductBacklog :exec
DELETE FROM backlog
WHERE project_id = $1 AND task_id = $2;

-- name: GetProductBacklog :many
SELECT b.project_id, b.task_id, b.sort_order
FROM backlog b
JOIN tasks t ON t.id = b.task_id
WHERE b.project_id = $1 AND t.deleted_at IS NULL
ORDER BY b.sort_order;

-- name: UpdateProductBacklogOrder :exec
UPDATE backlog
SET sort_order = $3
WHERE project_id = $1 AND task_id = $2;
