-- Task checklists and items

CREATE TABLE task_checklists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_checklists_task_id ON task_checklists(task_id);

CREATE TABLE checklist_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    checklist_id UUID NOT NULL REFERENCES task_checklists(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_checked BOOLEAN NOT NULL DEFAULT FALSE,
    "order" SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checklist_items_checklist_id ON checklist_items(checklist_id);

