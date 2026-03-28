-- =============================================================================
-- Projektus — Clean database schema
-- 30 tables, no reference tables (enums in Go code)
-- =============================================================================
SET client_encoding = 'UTF8';
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ======================== 1. Users & Auth ========================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(255),
    position VARCHAR(255),
    is_active BOOLEAN DEFAULT true NOT NULL,
    on_vacation BOOLEAN DEFAULT false NOT NULL,
    is_sick BOOLEAN DEFAULT false NOT NULL,
    alt_contact_channel VARCHAR(100),
    alt_contact_info VARCHAR(255),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_users_deleted ON users(deleted_at);

CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX idx_tokens_user ON tokens(user_id);
CREATE INDEX idx_tokens_hash ON tokens(token_hash);

CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX idx_pwd_history_user ON password_history(user_id, created_at DESC);

CREATE TABLE password_policy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    min_length INT NOT NULL DEFAULT 8,
    require_digits BOOLEAN DEFAULT true NOT NULL,
    require_lowercase BOOLEAN DEFAULT true NOT NULL,
    require_uppercase BOOLEAN DEFAULT true NOT NULL,
    require_special BOOLEAN DEFAULT true NOT NULL,
    notes TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_by UUID REFERENCES users(id)
);

-- ======================== 2. Rate Limiting ========================

CREATE TABLE login_attempts (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100),
    ip_address INET NOT NULL,
    success BOOLEAN NOT NULL,
    attempted_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX idx_login_ip ON login_attempts(ip_address, attempted_at);
CREATE INDEX idx_login_user ON login_attempts(username, attempted_at);

CREATE TABLE blocked_ips (
    ip_address INET PRIMARY KEY,
    blocked_until TIMESTAMPTZ NOT NULL
);

-- ======================== 3. Templates ========================

CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    project_type TEXT NOT NULL CHECK (project_type IN ('scrum','kanban'))
);

-- ======================== 4. Projects ========================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    project_type TEXT NOT NULL CHECK (project_type IN ('scrum','kanban')),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT DEFAULT 'active' NOT NULL CHECK (status IN ('active','archived','paused')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- ======================== 5. Roles & Permissions ========================

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope TEXT NOT NULL CHECK (scope IN ('system','template','project')),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_admin BOOLEAN DEFAULT false NOT NULL,
    sort_order INT DEFAULT 0 NOT NULL,
    CHECK (
        (scope = 'system' AND project_id IS NULL AND template_id IS NULL) OR
        (scope = 'project' AND project_id IS NOT NULL AND template_id IS NULL) OR
        (scope = 'template' AND project_id IS NULL AND template_id IS NOT NULL)
    )
);
CREATE INDEX idx_roles_project ON roles(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_roles_template ON roles(template_id) WHERE template_id IS NOT NULL;

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_code TEXT NOT NULL,
    access TEXT,
    PRIMARY KEY (role_id, permission_code)
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- ======================== 6. Members ========================

CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (project_id, user_id)
);

CREATE TABLE member_roles (
    member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (member_id, role_id)
);

-- ======================== 7. Boards ========================

CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_default BOOLEAN DEFAULT false NOT NULL,
    priority_type TEXT DEFAULT 'priority' NOT NULL,
    estimation_unit TEXT DEFAULT 'story_points' NOT NULL,
    swimlane_group_by TEXT DEFAULT '' NOT NULL,
    CHECK (
        (project_id IS NOT NULL AND template_id IS NULL) OR
        (project_id IS NULL AND template_id IS NOT NULL)
    )
);
CREATE INDEX idx_boards_project ON boards(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_boards_template ON boards(template_id) WHERE template_id IS NOT NULL;

CREATE TABLE columns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    system_type TEXT,
    wip_limit SMALLINT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_locked BOOLEAN DEFAULT false NOT NULL,
    note TEXT DEFAULT '' NOT NULL
);
CREATE INDEX idx_columns_board ON columns(board_id);

CREATE TABLE swimlanes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    wip_limit SMALLINT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    note TEXT DEFAULT '' NOT NULL
);
CREATE INDEX idx_swimlanes_board ON swimlanes(board_id);

CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    column_id UUID REFERENCES columns(id) ON DELETE CASCADE,
    swimlane_id UUID REFERENCES swimlanes(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    CHECK (
        (column_id IS NOT NULL AND swimlane_id IS NULL) OR
        (column_id IS NULL AND swimlane_id IS NOT NULL)
    )
);

-- ======================== 8. Fields (params + custom fields) ========================

CREATE TABLE fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL CHECK (kind IN ('project_param','board_field')),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    board_id UUID REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    field_type TEXT NOT NULL,
    is_system BOOLEAN DEFAULT false NOT NULL,
    is_required BOOLEAN DEFAULT false NOT NULL,
    sort_order INT DEFAULT 0 NOT NULL,
    options JSONB,
    value TEXT,
    CHECK (
        (kind = 'project_param' AND board_id IS NULL AND (project_id IS NOT NULL OR template_id IS NOT NULL)) OR
        (kind = 'board_field' AND board_id IS NOT NULL AND project_id IS NULL AND template_id IS NULL)
    )
);
CREATE INDEX idx_fields_project ON fields(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_fields_template ON fields(template_id) WHERE template_id IS NOT NULL;
CREATE INDEX idx_fields_board ON fields(board_id) WHERE board_id IS NOT NULL;

-- ======================== 9. Tasks ========================

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    executor_id UUID REFERENCES members(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    deadline TIMESTAMPTZ,
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE RESTRICT,
    swimlane_id UUID REFERENCES swimlanes(id) ON DELETE SET NULL,
    status_type TEXT DEFAULT 'initial' NOT NULL,
    deleted_at TIMESTAMPTZ,
    delete_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE (project_id, key)
);
CREATE INDEX idx_tasks_project ON tasks(project_id);
CREATE INDEX idx_tasks_column ON tasks(column_id);

CREATE TABLE task_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(50) NOT NULL,
    CHECK (task_id <> depends_on_task_id)
);

CREATE TABLE task_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    entered_at TIMESTAMPTZ NOT NULL,
    left_at TIMESTAMPTZ
);
CREATE INDEX idx_status_history_task ON task_status_history(task_id);

CREATE TABLE task_field_values (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    value_text TEXT,
    value_number NUMERIC,
    value_datetime TIMESTAMPTZ,
    value_json JSONB,
    PRIMARY KEY (task_id, field_id)
);

-- ======================== 10. Sprints & Backlog ========================

CREATE TABLE sprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    goal TEXT,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT DEFAULT 'planned' NOT NULL CHECK (status IN ('planned','active','completed')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX idx_sprints_project ON sprints(project_id);

CREATE TABLE sprint_tasks (
    sprint_id UUID NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    sort_order INT,
    PRIMARY KEY (sprint_id, task_id)
);

CREATE TABLE backlog (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID UNIQUE NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    sort_order INT NOT NULL,
    PRIMARY KEY (project_id, task_id)
);

-- ======================== 11. Meetings ========================

CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    meeting_type VARCHAR(100) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    location TEXT,
    status VARCHAR(20) DEFAULT 'active' NOT NULL CHECK (status IN ('active','cancelled'))
);
CREATE INDEX idx_meetings_time ON meetings(start_time, end_time);

CREATE TABLE meeting_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL,
    UNIQUE (meeting_id, user_id)
);

-- ======================== 12. Notifications ========================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    payload JSONB,
    is_read BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX idx_notif_user ON notifications(user_id, is_read, created_at DESC);

CREATE TABLE notification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    in_system BOOLEAN DEFAULT true NOT NULL,
    in_email BOOLEAN DEFAULT false NOT NULL,
    reminder_offset_minutes INT,
    UNIQUE (user_id, event_type)
);
