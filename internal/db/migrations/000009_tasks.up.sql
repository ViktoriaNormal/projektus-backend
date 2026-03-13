-- Tasks basic lifecycle

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key TEXT NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES project_members(id) ON DELETE RESTRICT,
    executor_id UUID REFERENCES project_members(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    deadline TIMESTAMPTZ,
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE RESTRICT,
    swimlane_id UUID REFERENCES swimlanes(id) ON DELETE SET NULL,
    delete_reason TEXT,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tasks_project_key UNIQUE (project_id, key)
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_column_id ON tasks(column_id);
CREATE INDEX idx_tasks_swimlane_id ON tasks(swimlane_id);

