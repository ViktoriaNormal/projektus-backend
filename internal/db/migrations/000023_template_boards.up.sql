-- Template boards and related tables
-- Encoding: UTF-8

-- Add updated_at column to project_templates
ALTER TABLE project_templates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Template boards
CREATE TABLE IF NOT EXISTS template_boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES project_templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    "order" INT NOT NULL DEFAULT 1,
    priority_type TEXT NOT NULL DEFAULT 'priority',
    estimation_unit TEXT NOT NULL DEFAULT 'story_points',
    swimlane_group_by TEXT NOT NULL DEFAULT ''
);

-- Template board columns
CREATE TABLE IF NOT EXISTS template_board_columns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES template_boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    system_type TEXT NOT NULL DEFAULT 'in_progress',
    wip_limit INT,
    "order" INT NOT NULL DEFAULT 1,
    is_locked BOOLEAN NOT NULL DEFAULT false
);

-- Template board swimlanes
CREATE TABLE IF NOT EXISTS template_board_swimlanes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES template_boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    wip_limit INT,
    "order" INT NOT NULL DEFAULT 1
);

-- Template board priority values
CREATE TABLE IF NOT EXISTS template_board_priority_values (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES template_boards(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    "order" INT NOT NULL DEFAULT 1
);

-- Template board custom fields
CREATE TABLE IF NOT EXISTS template_board_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES template_boards(id) ON DELETE CASCADE,
    code TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    field_type TEXT NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT false,
    is_required BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    "order" INT NOT NULL DEFAULT 1,
    options JSONB,
    config JSONB
);
