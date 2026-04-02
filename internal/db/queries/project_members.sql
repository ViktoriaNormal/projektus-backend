-- Project members

-- name: ListProjectMembers :many
SELECT m.id, m.project_id, m.user_id
FROM members m
WHERE m.project_id = $1
ORDER BY m.id ASC;

-- name: GetProjectMember :one
SELECT m.id, m.project_id, m.user_id
FROM members m
WHERE m.id = $1;

-- name: GetMemberByProjectAndUser :one
SELECT id, project_id, user_id
FROM members
WHERE project_id = $1 AND user_id = $2;

-- name: AddProjectMember :one
INSERT INTO members (project_id, user_id)
VALUES ($1, $2)
ON CONFLICT (project_id, user_id) DO NOTHING
RETURNING id, project_id, user_id;

-- name: RemoveProjectMember :exec
DELETE FROM members
WHERE id = $1;

-- name: ListProjectMembersByUser :many
SELECT m.id, m.project_id, m.user_id, p.name AS project_name
FROM members m
JOIN projects p ON p.id = m.project_id
WHERE m.user_id = $1
ORDER BY p.name;
