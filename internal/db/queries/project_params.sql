-- Project params (stored in unified `fields` table, kind='project_param')

-- name: ListProjectParams :many
SELECT id, project_id, name, description, field_type, is_system, is_required, sort_order, options, value
FROM fields
WHERE project_id = $1 AND kind = 'project_param'
ORDER BY sort_order ASC;

-- name: GetProjectParamByID :one
SELECT id, project_id, name, description, field_type, is_system, is_required, sort_order, options, value
FROM fields
WHERE id = $1 AND kind = 'project_param';

-- name: CreateProjectParam :one
INSERT INTO fields (kind, project_id, name, description, field_type, is_system, is_required, sort_order, options, value)
VALUES ('project_param', $1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, project_id, name, description, field_type, is_system, is_required, sort_order, options, value;

-- name: UpdateProjectParam :one
UPDATE fields
SET name = COALESCE(sqlc.narg('name'), name),
    is_required = COALESCE(sqlc.narg('is_required'), is_required),
    options = COALESCE(sqlc.narg('options'), options),
    value = sqlc.narg('value')
WHERE id = sqlc.arg('id') AND kind = 'project_param'
RETURNING id, project_id, name, description, field_type, is_system, is_required, sort_order, options, value;

-- name: DeleteProjectParamByID :exec
DELETE FROM fields WHERE id = $1 AND kind = 'project_param';

-- name: UpdateProjectParamOrder :exec
UPDATE fields SET sort_order = $2 WHERE id = $1 AND kind = 'project_param';
