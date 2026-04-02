CREATE TABLE checklists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX idx_checklists_task ON checklists(task_id);

CREATE TABLE checklist_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checklist_id UUID NOT NULL REFERENCES checklists(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_checked BOOLEAN DEFAULT FALSE NOT NULL,
    sort_order SMALLINT DEFAULT 0 NOT NULL
);
CREATE INDEX idx_checklist_items_checklist ON checklist_items(checklist_id);
