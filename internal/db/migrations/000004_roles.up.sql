-- Roles and permissions for workflow subsystem (system & project scopes)

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    scope TEXT NOT NULL, -- 'system' or 'project'
    project_id UUID NULL, -- FK будет добавлен на этапе проектов
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_scope_check CHECK (scope IN ('system', 'project')),
    CONSTRAINT roles_name_scope_project_unique UNIQUE (name, scope, project_id)
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS role_users (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, user_id)
);

-- Seed basic administrator role and permission
INSERT INTO permissions (code, description)
VALUES ('system.roles.manage', 'Управление системными ролями и правами')
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (name, description, scope, project_id)
VALUES ('Administrator', 'Системный администратор с полным доступом', 'system', NULL)
ON CONFLICT (name, scope, project_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'system.roles.manage'
WHERE r.name = 'Administrator' AND r.scope = 'system' AND r.project_id IS NULL
ON CONFLICT DO NOTHING;

