-- Swimlane configuration for Kanban boards

-- name: UpdateSwimlaneConfig :one
UPDATE swimlanes
SET source_type    = $2,
    custom_field_id = $3,
    value_mappings = $4,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: GetSwimlaneConfig :one
SELECT *
FROM swimlanes
WHERE id = $1;

-- name: GetSwimlanesWithConfig :many
SELECT *
FROM swimlanes
WHERE board_id = $1
ORDER BY "order";

