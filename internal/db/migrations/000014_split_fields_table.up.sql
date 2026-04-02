-- Migration 000014: Split `fields` table into `project_params` and `board_fields`.
-- Eliminates the `kind` discriminator column and complex CHECK constraints (2NF fix).

-- ============================================================
-- 1. Create project_params table
-- ============================================================
CREATE TABLE project_params (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    field_type TEXT NOT NULL,
    is_required BOOLEAN DEFAULT false NOT NULL,
    options JSONB,
    value TEXT,
    CHECK (
        (project_id IS NOT NULL AND template_id IS NULL) OR
        (project_id IS NULL AND template_id IS NOT NULL)
    )
);
CREATE INDEX idx_project_params_project ON project_params(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_project_params_template ON project_params(template_id) WHERE template_id IS NOT NULL;

-- ============================================================
-- 2. Create board_fields table
-- ============================================================
CREATE TABLE board_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    field_type TEXT NOT NULL,
    is_required BOOLEAN DEFAULT false NOT NULL,
    options JSONB
);
CREATE INDEX idx_board_fields_board ON board_fields(board_id);

-- ============================================================
-- 3. Migrate data from fields to new tables
-- ============================================================
INSERT INTO project_params (id, project_id, template_id, name, description, field_type, is_required, options, value)
SELECT id, project_id, template_id, name, description, field_type, is_required, options, value
FROM fields
WHERE kind = 'project_param';

INSERT INTO board_fields (id, board_id, name, description, field_type, is_required, options)
SELECT id, board_id, name, description, field_type, is_required, options
FROM fields
WHERE kind = 'board_field';

-- ============================================================
-- 4. Update task_field_values FK to point at board_fields
-- ============================================================
ALTER TABLE task_field_values DROP CONSTRAINT IF EXISTS task_field_values_field_id_fkey;
ALTER TABLE task_field_values ADD CONSTRAINT task_field_values_field_id_fkey
    FOREIGN KEY (field_id) REFERENCES board_fields(id) ON DELETE CASCADE;

-- ============================================================
-- 5. Drop old fields table
-- ============================================================
DROP TABLE fields;
