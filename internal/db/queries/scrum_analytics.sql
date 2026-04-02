-- Scrum analytics queries

-- name: GetSprintTasksForAnalytics :many
SELECT t.id, t.estimation, c.system_type AS column_system_type
FROM tasks t
JOIN sprint_tasks st ON st.task_id = t.id
LEFT JOIN columns c ON t.column_id = c.id
WHERE st.sprint_id = $1 AND t.deleted_at IS NULL;

-- name: GetSprintTaskStatusHistory :many
SELECT h.task_id, h.column_id, h.entered_at, h.left_at, c.system_type AS column_system_type
FROM task_status_history h
JOIN sprint_tasks st ON st.task_id = h.task_id
LEFT JOIN columns c ON h.column_id = c.id
WHERE st.sprint_id = $1
ORDER BY h.entered_at;
