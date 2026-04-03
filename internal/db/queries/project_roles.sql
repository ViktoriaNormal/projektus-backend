-- Roles for projects (scope='project', filtered by project_id)

-- name: ListProjRoleDefinitions :many
SELECT id, project_id, name, description, is_admin
FROM roles
WHERE project_id = $1;

-- name: GetProjRoleDefinitionByID :one
SELECT id, project_id, name, description, is_admin
FROM roles
WHERE id = $1;

-- name: CreateProjRoleDefinition :one
INSERT INTO roles (project_id, scope, name, description, is_admin)
VALUES ($1, 'project', $2, $3, $4)
RETURNING id, project_id, name, description, is_admin;

-- name: UpdateProjRoleDefinition :one
UPDATE roles
SET name = $2, description = $3
WHERE id = $1
RETURNING id, project_id, name, description, is_admin;

-- name: DeleteProjRoleDefinitionByID :exec
DELETE FROM roles WHERE id = $1;

-- name: CountProjRoleDefinitions :one
SELECT COUNT(*)::int AS count FROM roles WHERE project_id = $1;

-- name: CountProjRoleDefinitionMembers :one
SELECT COUNT(*)::int AS count
FROM member_roles mr
WHERE mr.role_id = $1;

-- name: GetProjectAdminRoleID :one
SELECT id FROM roles WHERE project_id = $1 AND is_admin = true LIMIT 1;

-- name: CountMembersWithRole :one
SELECT COUNT(*)::int AS count
FROM member_roles mr
JOIN members m ON m.id = mr.member_id
WHERE m.project_id = $1 AND mr.role_id = $2;

-- Role permissions (uses permission_code directly)

-- name: ListProjRoleDefPermissions :many
SELECT rp.role_id, rp.permission_code, rp.access
FROM role_permissions rp
WHERE rp.role_id = $1;

-- name: UpsertProjRoleDefPermission :exec
INSERT INTO role_permissions (role_id, permission_code, access)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, permission_code) DO UPDATE SET access = EXCLUDED.access;

-- name: DeleteProjRoleDefPermissionsByRoleID :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: GetMemberProjectPermissions :many
SELECT rp.permission_code, rp.access
FROM members m
JOIN member_roles mr ON mr.member_id = m.id
JOIN role_permissions rp ON rp.role_id = mr.role_id
WHERE m.project_id = $1 AND m.user_id = $2;
