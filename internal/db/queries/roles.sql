-- System roles and permissions

-- name: ListSystemRoles :many
SELECT id, name, description, scope, project_id, created_at, updated_at
FROM roles
WHERE scope = 'system'
ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description, scope, project_id, created_at, updated_at
FROM roles
WHERE id = $1;

-- name: CreateSystemRole :one
INSERT INTO roles (name, description, scope, project_id)
VALUES ($1, $2, 'system', NULL)
RETURNING id, name, description, scope, project_id, created_at, updated_at;

-- name: UpdateSystemRole :one
UPDATE roles
SET name = $2,
    description = $3,
    updated_at = NOW()
WHERE id = $1 AND scope = 'system'
RETURNING id, name, description, scope, project_id, created_at, updated_at;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = $1;

-- name: ListProjectRoles :many
SELECT id, name, description, scope, project_id, created_at, updated_at
FROM roles
WHERE scope = 'project' AND project_id = $1
ORDER BY name;

