-- Board custom fields (stored in unified `fields` table, kind='board_field')

-- name: ListBoardCustomFields :many
SELECT id, board_id, name, description, field_type, is_system, is_required, options
FROM fields
WHERE board_id = $1 AND kind = 'board_field';

-- name: GetBoardCustomFieldByID :one
SELECT id, board_id, name, description, field_type, is_system, is_required, options
FROM fields
WHERE id = $1 AND kind = 'board_field';

-- name: CreateBoardCustomField :one
INSERT INTO fields (kind, board_id, name, description, field_type, is_system, is_required, options)
VALUES ('board_field', $1, $2, $3, $4, $5, $6, $7)
RETURNING id, board_id, name, description, field_type, is_system, is_required, options;

-- name: UpdateBoardCustomField :one
UPDATE fields
SET name = $2, is_required = $3, options = $4
WHERE id = $1 AND kind = 'board_field'
RETURNING id, board_id, name, description, field_type, is_system, is_required, options;

-- name: DeleteBoardCustomFieldByID :exec
DELETE FROM fields WHERE id = $1 AND kind = 'board_field';
