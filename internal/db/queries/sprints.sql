-- Sprints basic CRUD

-- name: CreateSprint :one
INSERT INTO sprints (project_id, name, goal, start_date, end_date, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, name, goal, start_date, end_date, status, created_at, updated_at;

-- name: GetSprintByID :one
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE id = $1;

-- name: GetProjectSprints :many
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE project_id = $1
ORDER BY start_date DESC;

-- name: UpdateSprint :one
UPDATE sprints
SET name       = COALESCE(sqlc.narg('name'), name),
    goal       = sqlc.narg('goal'),
    start_date = COALESCE(sqlc.narg('start_date'), start_date),
    end_date   = COALESCE(sqlc.narg('end_date'), end_date),
    status     = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, project_id, name, goal, start_date, end_date, status, created_at, updated_at;

-- name: DeleteSprint :exec
DELETE FROM sprints
WHERE id = $1;

-- name: GetActiveSprint :one
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
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

-- name: GetNextPlannedSprint :one
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE project_id = $1 AND status = 'planned'
ORDER BY start_date ASC
LIMIT 1;

-- name: GetPlannedSprintsByProject :many
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE project_id = $1 AND status = 'planned'
ORDER BY start_date ASC;

-- name: GetNonCompletedSprintsByProject :many
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE project_id = $1 AND status IN ('planned', 'active')
ORDER BY start_date ASC;

-- name: GetCompletedSprintsByProject :many
SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at
FROM sprints
WHERE project_id = $1 AND status IN ('completed', 'active')
ORDER BY start_date ASC;
