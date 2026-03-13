-- Project templates

CREATE TABLE project_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    project_type TEXT NOT NULL, -- 'scrum' | 'kanban'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

