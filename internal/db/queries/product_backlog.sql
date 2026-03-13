-- Product backlog ordering

-- name: AddToProductBacklog :one
INSERT INTO product_backlog (project_id, task_id, "order")
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveFromProductBacklog :exec
DELETE FROM product_backlog
WHERE project_id = $1 AND task_id = $2;

-- name: GetProductBacklog :many
SELECT *
FROM product_backlog
WHERE project_id = $1
ORDER BY "order";

-- name: UpdateProductBacklogOrder :exec
UPDATE product_backlog
SET "order" = $3
WHERE project_id = $1 AND task_id = $2;

