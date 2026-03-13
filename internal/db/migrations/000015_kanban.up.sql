-- Kanban: classes of service, swimlane config, task status history, forecast cache

-- 1) Extend tasks with class_of_service and optional cached cycle time
ALTER TABLE tasks
    ADD COLUMN class_of_service TEXT,
    ADD COLUMN cycle_time_seconds BIGINT;

-- 2) Extend swimlanes with configuration for source type and custom field mapping
ALTER TABLE swimlanes
    ADD COLUMN source_type TEXT,
    ADD COLUMN custom_field_id UUID,
    ADD COLUMN value_mappings JSONB;

-- 3) Task status history for cycle time and CFD analytics
CREATE TABLE task_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    entered_at TIMESTAMPTZ NOT NULL,
    left_at TIMESTAMPTZ
);

CREATE INDEX idx_task_status_history_task_id ON task_status_history(task_id);
CREATE INDEX idx_task_status_history_column_id ON task_status_history(column_id);

-- 4) Kanban forecast cache for Monte Carlo results
CREATE TABLE kanban_forecast_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_item_count INTEGER NOT NULL,
    forecast_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_kanban_forecast_cache_project ON kanban_forecast_cache(project_id);
CREATE INDEX idx_kanban_forecast_cache_project_items ON kanban_forecast_cache(project_id, work_item_count);

