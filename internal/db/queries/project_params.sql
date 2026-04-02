-- Project params (custom only; system params generated from Go constants)

-- name: ListProjectParams :many
SELECT id, project_id, name, field_type, is_required, options, value
FROM project_params
WHERE project_id = $1;

-- name: GetProjectParamByID :one
SELECT id, project_id, name, field_type, is_required, options, value
FROM project_params
WHERE id = $1;

-- name: CreateProjectParam :one
INSERT INTO project_params (project_id, name, field_type, is_required, options, value)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, name, field_type, is_required, options, value;

-- name: UpdateProjectParam :one
UPDATE project_params
SET name = COALESCE(sqlc.narg('name'), name),
    is_required = COALESCE(sqlc.narg('is_required'), is_required),
    options = COALESCE(sqlc.narg('options'), options),
    value = sqlc.narg('value')
WHERE id = sqlc.arg('id')
RETURNING id, project_id, name, field_type, is_required, options, value;

-- name: DeleteProjectParamByID :exec
DELETE FROM project_params WHERE id = $1;
