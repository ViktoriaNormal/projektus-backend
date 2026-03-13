-- Scrum: sprints, sprint_tasks, product_backlog, story_points, backlog_type, is_sprint_backlog

-- 1) Extend tasks with story_points and backlog_type
ALTER TABLE tasks
    ADD COLUMN story_points INTEGER,
    ADD COLUMN backlog_type TEXT; -- 'product' | 'sprint' | NULL

CREATE INDEX idx_tasks_backlog_type ON tasks(backlog_type);

-- 2) Add is_sprint_backlog flag to columns (per board)
ALTER TABLE columns
    ADD COLUMN is_sprint_backlog BOOLEAN NOT NULL DEFAULT FALSE;

-- В рамках одной доски только одна колонка с is_sprint_backlog = TRUE
CREATE UNIQUE INDEX uq_columns_sprint_backlog_per_board
    ON columns(board_id)
    WHERE is_sprint_backlog IS TRUE;

-- 3) Sprints table
CREATE TABLE sprints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    goal TEXT,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'planned', -- planned | active | completed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sprints_project_id ON sprints(project_id);
CREATE INDEX idx_sprints_status ON sprints(status);

-- 4) Sprint tasks
CREATE TABLE sprint_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sprint_id UUID NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    "order" INTEGER,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sprint_task UNIQUE (sprint_id, task_id)
);

CREATE INDEX idx_sprint_tasks_sprint_id ON sprint_tasks(sprint_id);
CREATE INDEX idx_sprint_tasks_task_id ON sprint_tasks(task_id);

-- 5) Product backlog
CREATE TABLE product_backlog (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    "order" INTEGER NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_product_backlog_task UNIQUE (task_id)
);

CREATE INDEX idx_product_backlog_project_id ON product_backlog(project_id);

