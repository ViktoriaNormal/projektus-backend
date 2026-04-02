-- Migration 000013: Full normalization
-- Fixes: status_type redundancy, CHECK constraints, indexes, timezone, swimlane_group_by type

-- ============================================================
-- 1. Remove redundant status_type from tasks
--    (task status is derived from column.system_type via column_id)
-- ============================================================
ALTER TABLE tasks DROP COLUMN IF EXISTS status_type;

-- ============================================================
-- 2. Add CHECK constraints for TEXT enum columns
-- ============================================================

-- columns.system_type
ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_system_type_check;
ALTER TABLE columns ADD CONSTRAINT columns_system_type_check
    CHECK (system_type IS NULL OR system_type IN ('initial', 'in_progress', 'completed'));

-- task_dependencies.dependency_type
ALTER TABLE task_dependencies DROP CONSTRAINT IF EXISTS task_dependencies_type_check;
ALTER TABLE task_dependencies ADD CONSTRAINT task_dependencies_type_check
    CHECK (dependency_type IN ('blocks', 'blocked_by', 'related'));

-- meetings.status
ALTER TABLE meetings DROP CONSTRAINT IF EXISTS meetings_status_check;
ALTER TABLE meetings ADD CONSTRAINT meetings_status_check
    CHECK (status IN ('active', 'cancelled'));

-- meeting_participants.status
ALTER TABLE meeting_participants DROP CONSTRAINT IF EXISTS meeting_participants_status_check;
ALTER TABLE meeting_participants ADD CONSTRAINT meeting_participants_status_check
    CHECK (status IN ('pending', 'accepted', 'declined'));

-- role_permissions.access
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_access_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_access_check
    CHECK (access IN ('full', 'view', 'none'));

-- ============================================================
-- 3. Add CHECK constraint on task_field_values
--    Exactly one value column must be non-NULL
-- ============================================================
ALTER TABLE task_field_values DROP CONSTRAINT IF EXISTS task_field_values_one_value_check;
ALTER TABLE task_field_values ADD CONSTRAINT task_field_values_one_value_check
    CHECK (
        (CASE WHEN value_text IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN value_number IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN value_datetime IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN value_json IS NOT NULL THEN 1 ELSE 0 END) = 1
    );

-- ============================================================
-- 4. Fix users.blocked_until timezone
-- ============================================================
ALTER TABLE users ALTER COLUMN blocked_until TYPE TIMESTAMPTZ USING blocked_until AT TIME ZONE 'UTC';

-- ============================================================
-- 5. Add missing indexes
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_tasks_executor_id ON tasks (executor_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on ON task_dependencies (depends_on_task_id);
CREATE INDEX IF NOT EXISTS idx_meetings_project_start ON meetings (project_id, start_time);

-- ============================================================
-- 6. Add CHECK for wip_limit > 0 (columns and swimlanes)
-- ============================================================
ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_wip_limit_positive;
ALTER TABLE columns ADD CONSTRAINT columns_wip_limit_positive
    CHECK (wip_limit IS NULL OR wip_limit > 0);

ALTER TABLE swimlanes DROP CONSTRAINT IF EXISTS swimlanes_wip_limit_positive;
ALTER TABLE swimlanes ADD CONSTRAINT swimlanes_wip_limit_positive
    CHECK (wip_limit IS NULL OR wip_limit > 0);
