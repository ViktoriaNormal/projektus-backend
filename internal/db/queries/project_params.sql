-- Project params (stored in unified `fields` table, kind='project_param')

-- name: ListProjectParams :many
SELECT id, project_id, name, description, field_type, is_system, is_required, options, value
FROM fields
WHERE project_id = $1 AND kind = 'project_param';

-- name: GetProjectParamByID :one
SELECT id, project_id, name, description, field_type, is_system, is_required, options, value
FROM fields
WHERE id = $1 AND kind = 'project_param';

-- name: CreateProjectParam :one
INSERT INTO fields (kind, project_id, name, description, field_type, is_system, is_required, options, value)
VALUES ('project_param', $1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, project_id, name, description, field_type, is_system, is_required, options, value;

-- name: UpdateProjectParam :one
UPDATE fields
SET name = COALESCE(sqlc.narg('name'), name),
    is_required = COALESCE(sqlc.narg('is_required'), is_required),
    options = COALESCE(sqlc.narg('options'), options),
    value = sqlc.narg('value')
WHERE id = sqlc.arg('id') AND kind = 'project_param'
RETURNING id, project_id, name, description, field_type, is_system, is_required, options, value;

-- name: DeleteProjectParamByID :exec
DELETE FROM fields WHERE id = $1 AND kind = 'project_param';
