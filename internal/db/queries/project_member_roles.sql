-- Member roles

-- name: ListMemberRoles :many
SELECT r.id::text
FROM member_roles mr
JOIN roles r ON r.id = mr.role_id
WHERE mr.member_id = $1;

-- name: AddRoleToMember :exec
INSERT INTO member_roles (member_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteMemberRoles :exec
DELETE FROM member_roles
WHERE member_id = $1;

-- name: ListMemberRoleIDs :many
SELECT mr.role_id
FROM member_roles mr
WHERE mr.member_id = $1;
