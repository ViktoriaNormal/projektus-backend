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

-- name: UpdateTaskStoryPoints :exec
UPDATE tasks
SET story_points = $2
WHERE id = $1;

-- name: ListTasksByBacklogType :many
SELECT *
FROM tasks
WHERE project_id = $1
  AND backlog_type = $2
  AND deleted_at IS NULL
ORDER BY created_at DESC;


-- Watchers

-- name: AddTaskWatcher :one
INSERT INTO task_watchers (task_id, project_member_id)
VALUES ($1, $2)
ON CONFLICT (task_id, project_member_id) DO NOTHING
RETURNING *;

-- name: RemoveTaskWatcher :exec
DELETE FROM task_watchers
WHERE id = $1;

-- name: ListTaskWatchers :many
SELECT *
FROM task_watchers
WHERE task_id = $1;

-- Dependencies

-- name: AddTaskDependency :one
INSERT INTO task_dependencies (task_id, depends_on_task_id, dependency_type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveTaskDependency :exec
DELETE FROM task_dependencies
WHERE id = $1;

-- name: ListTaskDependencies :many
SELECT *
FROM task_dependencies
WHERE task_id = $1;

-- name: ListTaskDependants :many
SELECT *
FROM task_dependencies
WHERE depends_on_task_id = $1;

-- Checklists

-- name: CreateChecklist :one
INSERT INTO task_checklists (task_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: ListTaskChecklists :many
SELECT *
FROM task_checklists
WHERE task_id = $1
ORDER BY created_at;

-- name: CreateChecklistItem :one
INSERT INTO checklist_items (checklist_id, content, "order")
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListChecklistItems :many
SELECT *
FROM checklist_items
WHERE checklist_id = $1
ORDER BY "order";

-- name: UpdateChecklistItemStatus :one
UPDATE checklist_items
SET is_checked = $2
WHERE id = $1
RETURNING *;



