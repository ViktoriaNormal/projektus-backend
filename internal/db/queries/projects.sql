-- Projects

-- name: CreateProject :one
INSERT INTO projects (key, name, description, project_type, owner_id, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, key, name, description, project_type, owner_id, status, created_at;

-- name: GetProjectByID :one
SELECT id, key, name, description, project_type, owner_id, status, created_at
FROM projects
WHERE id = $1;

-- name: GetProjectByKey :one
SELECT id, key, name, description, project_type, owner_id, status, created_at
FROM projects
WHERE key = $1;

-- name: ListUserProjects :many
SELECT DISTINCT p.id, p.key, p.name, p.description, p.project_type, p.owner_id, p.status, p.created_at,
       u.full_name AS owner_full_name, u.avatar_url AS owner_avatar_url, u.email AS owner_email
FROM projects p
JOIN users u ON u.id = p.owner_id
LEFT JOIN members m ON m.project_id = p.id AND m.user_id = $1
WHERE (p.owner_id = $1 OR m.user_id IS NOT NULL)
  AND (sqlc.narg(status_filter)::text IS NULL OR p.status = sqlc.narg(status_filter))
  AND (sqlc.narg(type_filter)::text IS NULL OR p.project_type = sqlc.narg(type_filter))
  AND (sqlc.narg(search_query)::text IS NULL OR sqlc.narg(search_query)::text = '' OR (
       p.name ILIKE '%' || sqlc.narg(search_query) || '%'
       OR p.key ILIKE '%' || sqlc.narg(search_query) || '%'
       OR COALESCE(p.description, '') ILIKE '%' || sqlc.narg(search_query) || '%'
       OR u.full_name ILIKE '%' || sqlc.narg(search_query) || '%'))
ORDER BY p.created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    owner_id = COALESCE(sqlc.narg('owner_id'), owner_id)
WHERE id = sqlc.arg('id')
RETURNING id, key, name, description, project_type, owner_id, status, created_at;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
