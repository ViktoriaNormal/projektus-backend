-- Kanban analytics: CFD, Throughput, WIP

-- Task completion: when task first entered completed/cancelled column
-- name: GetTaskCompletedAt :many
SELECT
    tsh.task_id,
    MIN(tsh.entered_at) AS completed_at
FROM task_status_history tsh
JOIN columns c ON c.id = tsh.column_id
JOIN tasks t ON t.id = tsh.task_id
WHERE t.project_id = $1
  AND c.system_type IN ('completed', 'cancelled')
GROUP BY tsh.task_id;

-- CFD: per-date, per-column count of tasks in that column (for a board)
-- name: GetCfdColumnCountsByDate :many
WITH dates AS (
    SELECT generate_series($2::date, $3::date, '1 day'::interval)::date AS date
),
board_columns AS (
    SELECT id, name, "order"
    FROM columns
    WHERE columns.board_id = $4
),
task_completed AS (
    SELECT tsh.task_id, MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('completed', 'cancelled')
    GROUP BY tsh.task_id
),
task_column_on_date AS (
    SELECT DISTINCT ON (tsh.task_id, d.date)
        d.date,
        tsh.task_id,
        tsh.column_id
    FROM dates d
    CROSS JOIN task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    AND c.board_id = $4
    JOIN tasks t ON t.id = tsh.task_id AND t.project_id = $1
    WHERE t.deleted_at IS NULL
      AND tsh.entered_at <= d.date + interval '1 day'
      AND (tsh.left_at IS NULL OR tsh.left_at::date > d.date)
      AND t.created_at <= d.date + interval '1 day'
      AND NOT EXISTS (
          SELECT 1 FROM task_completed tc
          WHERE tc.task_id = tsh.task_id AND tc.completed_at::date <= d.date
      )
    ORDER BY tsh.task_id, d.date, tsh.entered_at DESC
)
SELECT
    tcd.date,
    bc.id AS column_id,
    bc.name AS column_name,
    bc."order" AS column_order,
    COUNT(tcd.task_id)::int AS task_count
FROM dates d
CROSS JOIN board_columns bc
LEFT JOIN task_column_on_date tcd ON tcd.date = d.date AND tcd.column_id = bc.id
GROUP BY d.date, bc.id, bc.name, bc."order"
ORDER BY d.date, bc."order";

-- Throughput: count of tasks completed per period (week or day), optional class_of_service
-- name: GetThroughputByPeriod :many
WITH task_completed AS (
    SELECT
        t.id AS task_id,
        t.class_of_service,
        MIN(tsh.entered_at) AS completed_at
    FROM tasks t
    JOIN task_status_history tsh ON tsh.task_id = t.id
    JOIN columns c ON c.id = tsh.column_id AND c.system_type IN ('completed', 'cancelled')
    WHERE t.project_id = $1
      AND t.deleted_at IS NULL
    GROUP BY t.id, t.class_of_service
)
SELECT
    date_trunc($4::text, tc.completed_at)::timestamp AS period_start,
    tc.class_of_service,
    COUNT(*)::int AS task_count
FROM task_completed tc
WHERE tc.completed_at BETWEEN $2::timestamptz AND $3::timestamptz
GROUP BY date_trunc($4::text, tc.completed_at), tc.class_of_service
ORDER BY period_start, tc.class_of_service;

-- Throughput simple (no group by class): for backward compatibility and simple charts
-- name: GetThroughputSimple :many
WITH task_completed AS (
    SELECT
        t.id AS task_id,
        MIN(tsh.entered_at) AS completed_at
    FROM tasks t
    JOIN task_status_history tsh ON tsh.task_id = t.id
    JOIN columns c ON c.id = tsh.column_id AND c.system_type IN ('completed', 'cancelled')
    WHERE t.project_id = $1
      AND t.deleted_at IS NULL
    GROUP BY t.id
)
SELECT
    (date_trunc('day', tc.completed_at)::date)::timestamp AS day_start,
    COUNT(*)::int AS task_count
FROM task_completed tc
WHERE tc.completed_at BETWEEN $2::timestamptz AND $3::timestamptz
GROUP BY date_trunc('day', tc.completed_at)
ORDER BY day_start;

-- WIP over time: for each date, count of tasks in progress (not in initial, not completed)
-- name: GetWipOverTime :many
WITH dates AS (
    SELECT generate_series($2::date, $3::date, '1 day'::interval)::date AS date
),
task_completed AS (
    SELECT tsh.task_id, MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('completed', 'cancelled')
    GROUP BY tsh.task_id
),
wip_per_day AS (
    SELECT
        d.date,
        COUNT(t.id)::int AS wip_count
    FROM dates d
    CROSS JOIN tasks t
    LEFT JOIN task_completed tc ON tc.task_id = t.id
    JOIN columns col ON col.id = t.column_id
    WHERE t.project_id = $1
      AND t.deleted_at IS NULL
      AND t.created_at <= d.date + interval '1 day'
      AND (tc.completed_at IS NULL OR tc.completed_at::date > d.date)
      AND col.system_type NOT IN ('initial', 'completed', 'cancelled')
    GROUP BY d.date
)
SELECT date, wip_count
FROM wip_per_day
ORDER BY date;

-- WIP over time with age (avg/max age in days)
-- name: GetWipWithAge :many
WITH dates AS (
    SELECT generate_series($2::date, $3::date, '1 day'::interval)::date AS date
),
task_completed AS (
    SELECT tsh.task_id, MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('completed', 'cancelled')
    GROUP BY tsh.task_id
),
task_in_progress_start AS (
    SELECT
        tsh.task_id,
        MIN(tsh.entered_at) AS started_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('in_progress', 'paused')
    GROUP BY tsh.task_id
),
daily_wip AS (
    SELECT
        d.date,
        t.id AS task_id,
        COALESCE(tips.started_at, t.created_at) AS work_started_at
    FROM dates d
    JOIN tasks t ON t.project_id = $1 AND t.deleted_at IS NULL
    LEFT JOIN task_completed tc ON tc.task_id = t.id
    JOIN columns col ON col.id = t.column_id
    LEFT JOIN task_in_progress_start tips ON tips.task_id = t.id
    WHERE t.created_at <= d.date + interval '1 day'
      AND (tc.completed_at IS NULL OR tc.completed_at::date > d.date)
      AND col.system_type NOT IN ('initial', 'completed', 'cancelled')
)
SELECT
    date,
    COUNT(*)::int AS wip_count,
    COALESCE(AVG(EXTRACT(EPOCH FROM (date::timestamp + interval '1 day' - work_started_at)) / 86400.0), 0)::float AS avg_wip_age_days,
    COALESCE(MAX(EXTRACT(EPOCH FROM (date::timestamp + interval '1 day' - work_started_at)) / 86400.0), 0)::float AS max_wip_age_days
FROM daily_wip
GROUP BY date
ORDER BY date;

-- Cycle Time: от первого входа в in_progress/paused до входа в completed/cancelled
-- name: GetCycleTimeScatterplot :many
WITH task_started AS (
    SELECT
        tsh.task_id,
        MIN(tsh.entered_at) AS started_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('in_progress', 'paused')
    GROUP BY tsh.task_id
),
task_completed_at AS (
    SELECT
        tsh.task_id,
        MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('completed', 'cancelled')
    GROUP BY tsh.task_id
)
SELECT
    t.id AS task_id,
    t.key AS task_key,
    t.class_of_service,
    tc.completed_at,
    (EXTRACT(EPOCH FROM (tc.completed_at - ts.started_at)) / 86400.0)::double precision AS cycle_time_days
FROM tasks t
JOIN task_started ts ON ts.task_id = t.id
JOIN task_completed_at tc ON tc.task_id = t.id
WHERE t.project_id = $1
  AND t.deleted_at IS NULL
  AND tc.completed_at >= ts.started_at
  AND tc.completed_at BETWEEN $2::timestamptz AND $3::timestamptz
  AND ($4::text IS NULL OR $4 = '' OR t.class_of_service = $4)
ORDER BY tc.completed_at;

-- Average Cycle Time по периодам (day/week/month)
-- name: GetAverageCycleTimeByPeriod :many
WITH task_started AS (
    SELECT tsh.task_id, MIN(tsh.entered_at) AS started_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('in_progress', 'paused')
    GROUP BY tsh.task_id
),
task_completed_at AS (
    SELECT tsh.task_id, MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type IN ('completed', 'cancelled')
    GROUP BY tsh.task_id
),
cycle_times AS (
    SELECT
        t.id AS task_id,
        t.class_of_service,
        tc.completed_at,
        (EXTRACT(EPOCH FROM (tc.completed_at - ts.started_at)) / 86400.0)::double precision AS cycle_time_days
    FROM tasks t
    JOIN task_started ts ON ts.task_id = t.id
    JOIN task_completed_at tc ON tc.task_id = t.id
    WHERE t.project_id = $1
      AND t.deleted_at IS NULL
      AND tc.completed_at >= ts.started_at
      AND tc.completed_at BETWEEN $2::timestamptz AND $3::timestamptz
      AND ($4::text IS NULL OR $4 = '' OR t.class_of_service = $4)
)
SELECT
    date_trunc($5::text, completed_at)::timestamp AS period_start,
    class_of_service,
    AVG(cycle_time_days)::float AS avg_cycle_time_days,
    COUNT(*)::int AS task_count
FROM cycle_times
GROUP BY date_trunc($5::text, completed_at), class_of_service
ORDER BY period_start, class_of_service;
