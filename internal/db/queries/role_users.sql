-- User role assignments (system roles)

-- name: ListUserSystemRoles :many
SELECT r.id, r.name, r.description, r.scope, r.is_admin
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.user_id = $1
  AND r.scope = 'system'
ORDER BY r.name;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($2, $1)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE role_id = $1 AND user_id = $2;

-- name: DeleteUserRoles :exec
DELETE FROM user_roles
WHERE user_id = $1;

-- name: DeleteUserSystemRoles :exec
DELETE FROM user_roles
WHERE user_id = $1
  AND role_id IN (SELECT id FROM roles WHERE scope = 'system');

-- name: UserHasSystemPermission :one
SELECT EXISTS (
    SELECT 1
    FROM user_roles ur
    JOIN roles r ON r.id = ur.role_id
    JOIN role_permissions rp ON rp.role_id = r.id
    WHERE ur.user_id = $1
      AND r.scope = 'system'
      AND rp.permission_code = $2
) AS has_permission;
