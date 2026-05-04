-- Kanban analytics queries

-- name: GetDefaultBoardForProject :one
SELECT id, name, estimation_unit
FROM boards
WHERE project_id = $1
ORDER BY is_default DESC, sort_order ASC
LIMIT 1;

-- name: GetBoardColumnsForAnalytics :many
SELECT id, name, system_type, wip_limit, sort_order
FROM columns
WHERE board_id = $1
ORDER BY sort_order;

-- name: GetProjectTaskHistoryForKanban :many
SELECT
    h.task_id,
    h.column_id,
    c.name AS column_name,
    c.system_type AS column_system_type,
    h.entered_at,
    h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
JOIN columns c ON h.column_id = c.id
WHERE t.project_id = $1
  AND t.board_id = $2
  AND t.deleted_at IS NULL
ORDER BY h.entered_at;

-- name: GetCompletedTasksForKanban :many
SELECT
    t.id AS task_id,
    t.key AS task_key,
    t.estimation,
    MIN(CASE WHEN c.system_type IN ('in_progress', 'paused') THEN h.entered_at END)::timestamptz AS started_at,
    MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END)::timestamptz AS completed_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
JOIN columns c ON h.column_id = c.id
WHERE t.project_id = $1
  AND t.board_id = $2
  AND t.deleted_at IS NULL
GROUP BY t.id, t.key, t.estimation
HAVING MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END) IS NOT NULL
  AND MIN(CASE WHEN c.system_type IN ('in_progress', 'paused') THEN h.entered_at END) IS NOT NULL;

-- name: GetCurrentWipCount :one
SELECT COUNT(*)::int AS wip_count
FROM tasks t
JOIN columns c ON t.column_id = c.id
WHERE t.project_id = $1
  AND t.board_id = $2
  AND t.deleted_at IS NULL
  AND c.system_type IN ('in_progress', 'paused');

-- name: GetWipTaskIDsForKanban :many
SELECT t.id
FROM tasks t
JOIN columns c ON t.column_id = c.id
WHERE t.project_id = $1
  AND t.board_id = $2
  AND t.deleted_at IS NULL
  AND c.system_type IN ('in_progress', 'paused');

-- name: GetWipAgeTasksForKanban :many
-- Возраст отсчитываем от входа в текущую рабочую колонку (открытая запись в истории);
-- при отсутствии открытой записи — от первого попадания в любую колонку in_progress/paused.
SELECT
    t.id AS task_id,
    t.key AS task_key,
    c.name AS column_name,
    COALESCE(
        (SELECT h.entered_at
         FROM task_status_history h
         WHERE h.task_id = t.id
           AND h.column_id = t.column_id
           AND h.left_at IS NULL
         ORDER BY h.entered_at DESC
         LIMIT 1),
        (SELECT MIN(h2.entered_at)
         FROM task_status_history h2
         JOIN columns c2 ON c2.id = h2.column_id
         WHERE h2.task_id = t.id
           AND c2.system_type IN ('in_progress', 'paused')),
        t.created_at
    )::timestamptz AS work_started_at
FROM tasks t
JOIN columns c ON t.column_id = c.id
WHERE t.project_id = $1
  AND t.board_id = $2
  AND t.deleted_at IS NULL
  AND c.system_type IN ('in_progress', 'paused');
