-- Projects

-- name: CreateProject :one
INSERT INTO projects (key, name, description, project_type, owner_id, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, key, name, description, project_type, owner_id, status, created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, key, name, description, project_type, owner_id, status, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: GetProjectByKey :one
SELECT id, key, name, description, project_type, owner_id, status, created_at, updated_at
FROM projects
WHERE key = $1;

-- name: ListUserProjects :many
SELECT id, key, name, description, project_type, owner_id, status, created_at, updated_at
FROM projects
WHERE owner_id = $1
  AND ($2::text IS NULL OR status = $2)
  AND ($3::text IS NULL OR project_type = $3)
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, key, name, description, project_type, owner_id, status, created_at, updated_at;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

