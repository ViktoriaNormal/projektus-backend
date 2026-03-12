-- User system roles

-- name: ListUserSystemRoles :many
SELECT r.id, r.name, r.description, r.scope, r.project_id, r.created_at, r.updated_at
FROM role_users ru
JOIN roles r ON r.id = ru.role_id
WHERE ru.user_id = $1
  AND r.scope = 'system'
ORDER BY r.name;

-- name: AssignRoleToUser :exec
INSERT INTO role_users (role_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM role_users
WHERE role_id = $1 AND user_id = $2;

-- name: DeleteUserRoles :exec
DELETE FROM role_users
WHERE user_id = $1;

-- name: UserHasSystemPermission :one
SELECT EXISTS (
    SELECT 1
    FROM role_users ru
    JOIN roles r ON r.id = ru.role_id
    JOIN role_permissions rp ON rp.role_id = r.id
    JOIN permissions p ON p.id = rp.permission_id
    WHERE ru.user_id = $1
      AND r.scope = 'system'
      AND p.code = $2
) AS has_permission;

