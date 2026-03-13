-- Revert Kanban extensions

DROP TABLE IF EXISTS kanban_forecast_cache;

DROP TABLE IF EXISTS task_status_history;

ALTER TABLE swimlanes
    DROP COLUMN IF EXISTS value_mappings,
    DROP COLUMN IF EXISTS custom_field_id,
    DROP COLUMN IF EXISTS source_type;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS cycle_time_seconds,
    DROP COLUMN IF EXISTS class_of_service;

