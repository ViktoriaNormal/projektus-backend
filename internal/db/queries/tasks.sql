-- Tasks basic CRUD

-- name: CreateTask :one
INSERT INTO tasks (key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, priority, estimation, board_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, deleted_at, created_at, priority, estimation, board_id;

-- name: GetTaskByID :one
SELECT t.id, t.key, t.project_id, t.owner_id, t.executor_id, t.name, t.description,
       t.deadline, t.column_id, t.swimlane_id, t.deleted_at, t.created_at,
       t.priority, t.estimation, t.board_id,
       c.name AS column_name, c.system_type AS column_system_type,
       m_owner.user_id AS owner_user_id,
       m_exec.user_id AS executor_user_id
FROM tasks t
LEFT JOIN columns c ON t.column_id = c.id
JOIN members m_owner ON m_owner.id = t.owner_id
LEFT JOIN members m_exec ON m_exec.id = t.executor_id
WHERE t.id = $1;

-- name: ListProjectTasks :many
SELECT t.id, t.key, t.project_id, t.owner_id, t.executor_id, t.name, t.description,
       t.deadline, t.column_id, t.swimlane_id, t.deleted_at, t.created_at,
       t.priority, t.estimation, t.board_id,
       c.name AS column_name, c.system_type AS column_system_type,
       m_owner.user_id AS owner_user_id,
       m_exec.user_id AS executor_user_id
FROM tasks t
LEFT JOIN columns c ON t.column_id = c.id
JOIN members m_owner ON m_owner.id = t.owner_id
LEFT JOIN members m_exec ON m_exec.id = t.executor_id
WHERE t.project_id = $1
  AND t.deleted_at IS NULL
ORDER BY t.created_at DESC;

-- name: SearchTasks :many
SELECT DISTINCT t.id, t.key, t.project_id, t.owner_id, t.executor_id, t.name, t.description,
       t.deadline, t.column_id, t.swimlane_id, t.deleted_at, t.created_at,
       t.priority, t.estimation, t.board_id,
       c.name AS column_name, c.system_type AS column_system_type,
       m_owner.user_id AS owner_user_id,
       m_exec.user_id AS executor_user_id
FROM tasks t
LEFT JOIN columns c ON t.column_id = c.id
JOIN members m_owner ON m_owner.id = t.owner_id
LEFT JOIN members m_exec ON m_exec.id = t.executor_id
LEFT JOIN task_watchers tw ON tw.task_id = t.id
LEFT JOIN members m_watch ON m_watch.id = tw.member_id
WHERE (m_owner.user_id = sqlc.arg('user_id') OR m_exec.user_id = sqlc.arg('user_id') OR m_watch.user_id = sqlc.arg('user_id'))
  AND (sqlc.narg('project_id')::uuid IS NULL OR t.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('column_id')::uuid IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND t.deleted_at IS NULL
ORDER BY t.created_at DESC;

-- name: UpdateTask :one
UPDATE tasks
SET name        = COALESCE(sqlc.narg('name'), name),
    description = sqlc.narg('description'),
    deadline    = sqlc.narg('deadline'),
    executor_id = sqlc.narg('executor_id'),
    column_id   = COALESCE(sqlc.narg('column_id'), column_id),
    swimlane_id = sqlc.narg('swimlane_id'),
    priority    = sqlc.narg('priority'),
    estimation  = sqlc.narg('estimation')
WHERE id = sqlc.arg('id')
RETURNING id, key, project_id, owner_id, executor_id, name, description, deadline, column_id, swimlane_id, deleted_at, created_at, priority, estimation, board_id;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at = NOW()
WHERE id = $1;

-- name: AssignColumnToTask :exec
UPDATE tasks SET column_id = $2 WHERE id = $1;

-- name: ClearColumnFromTask :exec
UPDATE tasks SET column_id = NULL WHERE id = $1;

-- name: ListSprintTasksWithoutColumn :many
SELECT t.id, t.board_id
FROM tasks t
JOIN sprint_tasks st ON st.task_id = t.id
WHERE st.sprint_id = $1 AND t.column_id IS NULL;

-- name: ListProjectTaskKeys :many
SELECT key
FROM tasks
WHERE project_id = $1;

-- Dependencies

-- name: AddTaskDependency :one
INSERT INTO task_dependencies (task_id, depends_on_task_id, dependency_type)
VALUES ($1, $2, $3)
RETURNING id, task_id, depends_on_task_id, dependency_type;

-- name: GetTaskDependencyByID :one
SELECT id, task_id, depends_on_task_id, dependency_type
FROM task_dependencies
WHERE id = $1;

-- name: RemoveTaskDependency :exec
DELETE FROM task_dependencies
WHERE id = $1;

-- name: RemoveInverseDependency :exec
DELETE FROM task_dependencies
WHERE task_id = $1 AND depends_on_task_id = $2;

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
