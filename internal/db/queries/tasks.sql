-- Tasks basic CRUD

-- name: CreateTask :one
INSERT INTO tasks (key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetTaskByID :one
SELECT *
FROM tasks
WHERE id = $1;

-- name: ListProjectTasks :many
SELECT *
FROM tasks
WHERE project_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SearchTasks :many
SELECT *
FROM tasks
WHERE ($1::uuid IS NULL OR project_id = $1)
  AND ($2::uuid IS NULL OR owner_id = $2)
  AND ($3::uuid IS NULL OR executor_id = $3)
  AND ($4::uuid IS NULL OR column_id = $4)
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateTask :one
UPDATE tasks
SET name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    deadline    = COALESCE(sqlc.narg('deadline'), deadline),
    executor_id = COALESCE(sqlc.narg('executor_id'), executor_id),
    column_id   = COALESCE(sqlc.narg('column_id'), column_id),
    swimlane_id = COALESCE(sqlc.narg('swimlane_id'), swimlane_id),
    updated_at  = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at   = NOW(),
    delete_reason = $2,
    updated_at   = NOW()
WHERE id = $1;

-- name: ListProjectTaskKeys :many
SELECT key
FROM tasks
WHERE project_id = $1;

