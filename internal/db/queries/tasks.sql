-- Tasks basic CRUD

-- name: CreateTask :one
INSERT INTO tasks (key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, status_type, deleted_at, delete_reason, created_at;

-- name: GetTaskByID :one
SELECT id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, status_type, deleted_at, delete_reason, created_at
FROM tasks
WHERE id = $1;

-- name: ListProjectTasks :many
SELECT id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, status_type, deleted_at, delete_reason, created_at
FROM tasks
WHERE project_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SearchTasks :many
SELECT id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, status_type, deleted_at, delete_reason, created_at
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
    description = sqlc.narg('description'),
    deadline    = sqlc.narg('deadline'),
    executor_id = sqlc.narg('executor_id'),
    column_id   = COALESCE(sqlc.narg('column_id'), column_id),
    swimlane_id = sqlc.narg('swimlane_id')
WHERE id = sqlc.arg('id')
RETURNING id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, status_type, deleted_at, delete_reason, created_at;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at   = NOW(),
    delete_reason = $2
WHERE id = $1;

-- name: ListProjectTaskKeys :many
SELECT key
FROM tasks
WHERE project_id = $1;

-- Dependencies

-- name: AddTaskDependency :one
INSERT INTO task_dependencies (task_id, depends_on_task_id, dependency_type)
VALUES ($1, $2, $3)
RETURNING id, task_id, depends_on_task_id, dependency_type;

-- name: RemoveTaskDependency :exec
DELETE FROM task_dependencies
WHERE id = $1;

-- name: ListTaskDependencies :many
SELECT id, task_id, depends_on_task_id, dependency_type
FROM task_dependencies
WHERE task_id = $1;

-- name: ListTaskDependants :many
SELECT id, task_id, depends_on_task_id, dependency_type
FROM task_dependencies
WHERE depends_on_task_id = $1;

-- Task field values

-- name: UpsertTaskFieldValue :exec
INSERT INTO task_field_values (task_id, field_id, value_text, value_number, value_datetime, value_json)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (task_id, field_id) DO UPDATE
SET value_text = EXCLUDED.value_text,
    value_number = EXCLUDED.value_number,
    value_datetime = EXCLUDED.value_datetime,
    value_json = EXCLUDED.value_json;

-- name: GetTaskFieldValues :many
SELECT task_id, field_id, value_text, value_number, value_datetime, value_json
FROM task_field_values
WHERE task_id = $1;

-- name: DeleteTaskFieldValue :exec
DELETE FROM task_field_values
WHERE task_id = $1 AND field_id = $2;
