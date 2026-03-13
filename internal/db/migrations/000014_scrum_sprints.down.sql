ALTER TABLE tasks
    DROP COLUMN IF EXISTS story_points,
    DROP COLUMN IF EXISTS backlog_type;

ALTER TABLE columns
    DROP COLUMN IF EXISTS is_sprint_backlog;

DROP INDEX IF EXISTS uq_columns_sprint_backlog_per_board;
DROP INDEX IF EXISTS idx_tasks_backlog_type;
DROP INDEX IF EXISTS idx_product_backlog_project_id;
DROP INDEX IF EXISTS idx_sprint_tasks_task_id;
DROP INDEX IF EXISTS idx_sprint_tasks_sprint_id;
DROP INDEX IF EXISTS idx_sprints_status;
DROP INDEX IF EXISTS idx_sprints_project_id;

DROP TABLE IF EXISTS product_backlog;
DROP TABLE IF EXISTS sprint_tasks;
DROP TABLE IF EXISTS sprints;

