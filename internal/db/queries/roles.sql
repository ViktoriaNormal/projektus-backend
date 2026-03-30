-- System roles (scope='system' in unified roles table)

-- name: ListSystemRoles :many
SELECT id, name, description, scope, is_admin
FROM roles
WHERE scope = 'system';

-- name: GetRoleByID :one
SELECT id, name, description, scope, is_admin
FROM roles
WHERE id = $1;

-- name: CreateSystemRole :one
INSERT INTO roles (name, description, scope)
VALUES ($1, $2, 'system')
RETURNING id, name, description, scope, is_admin;

-- name: UpdateSystemRole :one
UPDATE roles
SET name = $2, description = $3
WHERE id = $1 AND scope = 'system'
RETURNING id, name, description, scope, is_admin;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: ListProjectRoles :many
SELECT id, name, description, scope, project_id
FROM roles
WHERE scope = 'project' AND project_id = $1;

-- name: CreateProjectRole :one
INSERT INTO roles (name, description, scope, project_id)
VALUES ($1, $2, 'project', $3)
RETURNING id, name, description, scope, project_id;

-- Permissions for roles (permission_code directly in role_permissions)

-- name: ListRolePermissions :many
SELECT rp.role_id, rp.permission_code, rp.access
FROM role_permissions rp
WHERE rp.role_id = $1
ORDER BY rp.permission_code;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_code, access)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, permission_code) DO UPDATE SET access = EXCLUDED.access;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_code = $2;

-- name: RemoveAllPermissionsFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1;
