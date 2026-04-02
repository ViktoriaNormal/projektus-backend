-- Board custom fields (system fields generated from Go constants)

-- name: ListBoardCustomFields :many
SELECT id, board_id, name, field_type, is_required, options
FROM board_fields
WHERE board_id = $1;

-- name: GetBoardCustomFieldByID :one
SELECT id, board_id, name, field_type, is_required, options
FROM board_fields
WHERE id = $1;

-- name: CreateBoardCustomField :one
INSERT INTO board_fields (board_id, name, field_type, is_required, options)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, board_id, name, field_type, is_required, options;

-- name: UpdateBoardCustomField :one
UPDATE board_fields
SET name = $2, is_required = $3, options = $4
WHERE id = $1
RETURNING id, board_id, name, field_type, is_required, options;

-- name: DeleteBoardCustomFieldByID :exec
DELETE FROM board_fields WHERE id = $1;
