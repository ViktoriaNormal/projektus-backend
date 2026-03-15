-- Scrum analytics: velocity and burndown

-- name: GetVelocityData :many
WITH sprint_tasks_with_story AS (
    SELECT
        s.id        AS sprint_id,
        s.name      AS sprint_name,
        s.start_date,
        s.end_date,
        t.id        AS task_id,
        t.story_points
    FROM sprints s
    LEFT JOIN sprint_tasks st ON st.sprint_id = s.id
    LEFT JOIN tasks t ON t.id = st.task_id
    WHERE s.project_id = $1
      AND s.status = 'completed'
),
task_completion AS (
    SELECT
        tsh.task_id,
        MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type = 'completed'
    GROUP BY tsh.task_id
)
SELECT
    st.sprint_id,
    st.sprint_name,
    st.start_date,
    st.end_date,
    COALESCE(SUM(CASE WHEN st.task_id IS NOT NULL AND st.story_points IS NOT NULL THEN st.story_points ELSE 0 END), 0)::int AS committed_points,
    COALESCE(SUM(CASE
                     WHEN tc.completed_at IS NOT NULL
                          AND tc.completed_at::date <= st.end_date
                          AND st.story_points IS NOT NULL
                     THEN st.story_points
                     ELSE 0
                 END), 0)::int AS completed_points
FROM sprint_tasks_with_story st
LEFT JOIN task_completion tc ON tc.task_id = st.task_id
GROUP BY st.sprint_id, st.sprint_name, st.start_date, st.end_date
ORDER BY st.end_date;


-- name: GetBurndownData :many
WITH sprint AS (
    SELECT id AS sprint_id, start_date, end_date
    FROM sprints s
    WHERE s.id = $1
),
days_in_sprint AS (
    SELECT generate_series(
               (SELECT start_date FROM sprint),
               (SELECT end_date FROM sprint),
               '1 day'::interval
           )::date AS day
),
sprint_tasks_data AS (
    SELECT
        t.id           AS task_id,
        t.story_points
    FROM sprint_tasks st
    JOIN tasks t ON t.id = st.task_id
    WHERE st.sprint_id = $1
),
task_completion AS (
    SELECT
        tsh.task_id,
        MIN(tsh.entered_at) AS completed_at
    FROM task_status_history tsh
    JOIN columns c ON c.id = tsh.column_id
    WHERE c.system_type = 'completed'
    GROUP BY tsh.task_id
)
SELECT
    d.day,
    COALESCE(SUM(
        CASE
            WHEN sd.task_id IS NOT NULL
                 AND (tc.completed_at IS NULL OR tc.completed_at::date > d.day)
                 AND sd.story_points IS NOT NULL
            THEN sd.story_points
            ELSE 0
        END
    ), 0)::int AS remaining_points
FROM days_in_sprint d
LEFT JOIN sprint_tasks_data sd ON TRUE
LEFT JOIN task_completion tc ON tc.task_id = sd.task_id
GROUP BY d.day
ORDER BY d.day;

