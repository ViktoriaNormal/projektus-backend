-- Projects

-- name: CreateProject :one
INSERT INTO projects (key, name, description, project_type, owner_id, status, sprint_duration_weeks, incomplete_tasks_action)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, key, name, description, project_type, owner_id, status, created_at, sprint_duration_weeks, incomplete_tasks_action, deleted_at;

-- name: GetProjectByID :one
SELECT id, key, name, description, project_type, owner_id, status, created_at, sprint_duration_weeks, incomplete_tasks_action, deleted_at
FROM projects
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetProjectByKey :one
SELECT id, key, name, description, project_type, owner_id, status, created_at, sprint_duration_weeks, incomplete_tasks_action, deleted_at
FROM projects
WHERE key = $1
  AND deleted_at IS NULL;

-- name: ListUserProjects :many
SELECT DISTINCT p.id, p.key, p.name, p.description, p.project_type, p.owner_id, p.status, p.created_at,
       u.full_name AS owner_full_name, u.avatar_url AS owner_avatar_url, u.email AS owner_email
FROM projects p
JOIN users u ON u.id = p.owner_id
LEFT JOIN members m ON m.project_id = p.id AND m.user_id = $1
WHERE p.deleted_at IS NULL
  AND (p.owner_id = $1 OR m.user_id IS NOT NULL)
  AND (sqlc.narg(status_filter)::text IS NULL OR p.status = sqlc.narg(status_filter))
  AND (sqlc.narg(type_filter)::text IS NULL OR p.project_type = sqlc.narg(type_filter))
  AND (sqlc.narg(search_query)::text IS NULL OR sqlc.narg(search_query)::text = '' OR (
       p.name ILIKE '%' || sqlc.narg(search_query) || '%'
       OR p.key ILIKE '%' || sqlc.narg(search_query) || '%'
       OR COALESCE(p.description, '') ILIKE '%' || sqlc.narg(search_query) || '%'
       OR u.full_name ILIKE '%' || sqlc.narg(search_query) || '%'))
ORDER BY p.created_at DESC;

-- name: ListAllProjects :many
SELECT p.id, p.key, p.name, p.description, p.project_type, p.owner_id, p.status, p.created_at,
       u.full_name AS owner_full_name, u.avatar_url AS owner_avatar_url, u.email AS owner_email
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE p.deleted_at IS NULL
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
    description = sqlc.narg('description'),
    status = COALESCE(sqlc.narg('status'), status),
    owner_id = COALESCE(sqlc.narg('owner_id'), owner_id),
    sprint_duration_weeks = COALESCE(sqlc.narg('sprint_duration_weeks'), sprint_duration_weeks),
    incomplete_tasks_action = COALESCE(sqlc.narg('incomplete_tasks_action'), incomplete_tasks_action)
WHERE id = sqlc.arg('id')
  AND deleted_at IS NULL
RETURNING id, key, name, description, project_type, owner_id, status, created_at, sprint_duration_weeks, incomplete_tasks_action, deleted_at;

-- name: SoftDeleteProject :exec
UPDATE projects
SET deleted_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
