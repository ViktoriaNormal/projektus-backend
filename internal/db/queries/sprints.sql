-- Sprints basic CRUD

-- name: CreateSprint :one
INSERT INTO sprints (project_id, name, goal, start_date, end_date, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSprintByID :one
SELECT *
FROM sprints
WHERE id = $1;

-- name: GetProjectSprints :many
SELECT *
FROM sprints
WHERE project_id = $1
ORDER BY start_date DESC;

-- name: UpdateSprint :one
UPDATE sprints
SET name       = COALESCE(sqlc.narg('name'), name),
    goal       = COALESCE(sqlc.narg('goal'), goal),
    start_date = COALESCE(sqlc.narg('start_date'), start_date),
    end_date   = COALESCE(sqlc.narg('end_date'), end_date),
    status     = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSprint :exec
DELETE FROM sprints
WHERE id = $1;

-- name: GetActiveSprint :one
SELECT *
FROM sprints
WHERE project_id = $1
  AND status = 'active'
LIMIT 1;

-- name: UpdateSprintStatuses :exec
UPDATE sprints
SET status = CASE
                 WHEN CURRENT_DATE < start_date THEN 'planned'
                 WHEN CURRENT_DATE > end_date THEN 'completed'
                 ELSE 'active'
             END,
    updated_at = NOW();

