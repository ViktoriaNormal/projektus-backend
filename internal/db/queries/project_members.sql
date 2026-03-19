-- Project members

-- name: ListProjectMembers :many
SELECT pm.id,
       pm.project_id,
       pm.user_id,
       pm.created_at,
       pm.updated_at
FROM project_members pm
WHERE pm.project_id = $1
ORDER BY pm.created_at ASC;

-- name: GetProjectMember :one
SELECT pm.id,
       pm.project_id,
       pm.user_id,
       pm.created_at,
       pm.updated_at
FROM project_members pm
WHERE pm.id = $1;

-- name: AddProjectMember :one
INSERT INTO project_members (project_id, user_id)
VALUES ($1, $2)
ON CONFLICT (project_id, user_id) DO NOTHING
RETURNING id, project_id, user_id, created_at, updated_at;

-- name: RemoveProjectMember :exec
DELETE FROM project_members
WHERE id = $1;

-- name: ListProjectMembersByUser :many
SELECT pm.id,
       pm.project_id,
       pm.user_id,
       p.name AS project_name,
       pm.created_at,
       pm.updated_at
FROM project_members pm
JOIN projects p ON p.id = pm.project_id
WHERE pm.user_id = $1
ORDER BY p.name;

