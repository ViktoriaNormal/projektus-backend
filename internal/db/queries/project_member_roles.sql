-- Project member roles

-- name: ListMemberRoles :many
SELECT r.name
FROM project_member_roles pmr
JOIN roles r ON r.id = pmr.role_id
WHERE pmr.project_member_id = $1
ORDER BY r.name;

-- name: AddRoleToMember :exec
INSERT INTO project_member_roles (project_member_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteMemberRoles :exec
DELETE FROM project_member_roles
WHERE project_member_id = $1;

-- name: ListMemberRoleIDs :many
SELECT pmr.role_id
FROM project_member_roles pmr
WHERE pmr.project_member_id = $1;

