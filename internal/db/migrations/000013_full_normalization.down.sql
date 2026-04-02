-- Reverse normalization changes
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS status_type TEXT DEFAULT 'initial' NOT NULL;

ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_system_type_check;
ALTER TABLE task_dependencies DROP CONSTRAINT IF EXISTS task_dependencies_type_check;
ALTER TABLE meetings DROP CONSTRAINT IF EXISTS meetings_status_check;
ALTER TABLE meeting_participants DROP CONSTRAINT IF EXISTS meeting_participants_status_check;
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_access_check;
ALTER TABLE task_field_values DROP CONSTRAINT IF EXISTS task_field_values_one_value_check;
ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_wip_limit_positive;
ALTER TABLE swimlanes DROP CONSTRAINT IF EXISTS swimlanes_wip_limit_positive;

ALTER TABLE users ALTER COLUMN blocked_until TYPE TIMESTAMP USING blocked_until;

DROP INDEX IF EXISTS idx_tasks_executor_id;
DROP INDEX IF EXISTS idx_task_dependencies_depends_on;
DROP INDEX IF EXISTS idx_meetings_project_start;
