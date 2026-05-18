-- =============================================================================
-- Migration 040: привязка багов MOBAPP к спринтам (sprint_tasks)
--
-- В Scrum-проекте доска показывает только задачи текущего спринта.
-- Баг-трекер был заполнен в tasks, но без sprint_tasks — поэтому UI пустой.
-- =============================================================================

SET client_encoding = 'UTF8';

INSERT INTO sprint_tasks (sprint_id, task_id, sort_order)
SELECT
    mapped.sprint_id,
    mapped.task_id,
    ROW_NUMBER() OVER (
        PARTITION BY mapped.sprint_id
        ORDER BY mapped.created_at, mapped.task_id
    )::int AS sort_order
FROM (
    SELECT
        t.id AS task_id,
        t.created_at,
        COALESCE(ds.sprint_id, act.sprint_id, lst.sprint_id) AS sprint_id
    FROM tasks t
    JOIN boards b ON b.id = t.board_id AND b.is_default = false
    JOIN projects p ON p.id = t.project_id AND p.key = 'MOBAPP'
    LEFT JOIN LATERAL (
        SELECT s.id AS sprint_id
        FROM sprints s
        WHERE s.project_id = t.project_id
          AND t.created_at::date >= s.start_date
          AND t.created_at::date <= s.end_date
        ORDER BY s.start_date
        LIMIT 1
    ) ds ON true
    LEFT JOIN LATERAL (
        SELECT s.id AS sprint_id
        FROM sprints s
        WHERE s.project_id = t.project_id
          AND s.status = 'active'
        ORDER BY s.start_date DESC
        LIMIT 1
    ) act ON true
    LEFT JOIN LATERAL (
        SELECT s.id AS sprint_id
        FROM sprints s
        WHERE s.project_id = t.project_id
        ORDER BY s.end_date DESC
        LIMIT 1
    ) lst ON true
    WHERE t.deleted_at IS NULL
      AND NOT EXISTS (
          SELECT 1 FROM sprint_tasks st WHERE st.task_id = t.id
      )
) mapped
WHERE mapped.sprint_id IS NOT NULL
ON CONFLICT (sprint_id, task_id) DO NOTHING;
