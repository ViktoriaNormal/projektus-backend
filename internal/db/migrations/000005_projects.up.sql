-- Projects lifecycle and basic metadata

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    project_type TEXT NOT NULL, -- 'scrum' | 'kanban'
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'active', -- 'active' | 'archived' | 'paused'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_owner_status ON projects (owner_id, status);

ALTER TABLE roles
    ADD CONSTRAINT fk_roles_project
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

