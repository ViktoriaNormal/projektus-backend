-- Permissions

-- name: ListPermissions :many
SELECT id, code, description, created_at
FROM permissions
ORDER BY code;

-- name: GetPermissionByCode :one
SELECT id, code, description, created_at
FROM permissions
WHERE code = $1;

-- name: CreatePermission :one
INSERT INTO permissions (code, description)
VALUES ($1, $2)
RETURNING id, code, description, created_at;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = $1;

-- name: ListRolePermissions :many
SELECT p.id, p.code, p.description, p.created_at
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = $1
ORDER BY p.code;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;

