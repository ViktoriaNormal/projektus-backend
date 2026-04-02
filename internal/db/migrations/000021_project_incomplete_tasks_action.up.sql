ALTER TABLE projects ADD COLUMN incomplete_tasks_action TEXT NOT NULL DEFAULT 'backlog'
    CHECK (incomplete_tasks_action IN ('backlog', 'next_sprint'));
