-- Task status history for cycle time analytics

-- name: RecordTaskStatusChange :one
INSERT INTO task_status_history (task_id, column_id, entered_at, left_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTaskCycleTimes :many
SELECT
    h.task_id,
    EXTRACT(EPOCH FROM (MAX(h.left_at) - MIN(h.entered_at))) AS cycle_time_seconds
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = $1
  AND h.left_at IS NOT NULL
GROUP BY h.task_id;

-- name: GetTaskStatusHistory :many
SELECT *
FROM task_status_history
WHERE task_id = $1
ORDER BY entered_at;

-- name: GetCompletedTasksCycleTime :many
SELECT
    h.task_id,
    EXTRACT(EPOCH FROM (MAX(h.left_at) - MIN(h.entered_at))) AS cycle_time_seconds,
    MAX(h.left_at) AS completed_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = $1
  AND h.left_at IS NOT NULL
GROUP BY h.task_id;

