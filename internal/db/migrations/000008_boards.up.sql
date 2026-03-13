-- Boards, columns, swimlanes, notes

CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES project_templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    "order" SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_boards_project_or_template
        CHECK (
            (project_id IS NOT NULL AND template_id IS NULL) OR
            (project_id IS NULL AND template_id IS NOT NULL)
        )
);

CREATE INDEX idx_boards_project_id ON boards(project_id);
CREATE INDEX idx_boards_template_id ON boards(template_id);

CREATE TABLE columns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    system_type TEXT, -- 'initial', 'in_progress', 'paused', 'completed', 'cancelled'
    wip_limit SMALLINT,
    "order" SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_columns_board_id ON columns(board_id);

CREATE TABLE swimlanes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    wip_limit SMALLINT,
    "order" SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_swimlanes_board_id ON swimlanes(board_id);

CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    column_id UUID REFERENCES columns(id) ON DELETE CASCADE,
    swimlane_id UUID REFERENCES swimlanes(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notes_column_or_swimlane
        CHECK (
            (column_id IS NOT NULL AND swimlane_id IS NULL) OR
            (column_id IS NULL AND swimlane_id IS NOT NULL)
        )
);

