-- Reverse: recreate fields table and merge data back
CREATE TABLE fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL CHECK (kind IN ('project_param','board_field')),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    board_id UUID REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    field_type TEXT NOT NULL,
    is_required BOOLEAN DEFAULT false NOT NULL,
    options JSONB,
    value TEXT,
    CHECK (
        (kind = 'project_param' AND board_id IS NULL AND (project_id IS NOT NULL OR template_id IS NOT NULL)) OR
        (kind = 'board_field' AND board_id IS NOT NULL AND project_id IS NULL AND template_id IS NULL)
    )
);

INSERT INTO fields (id, kind, project_id, template_id, name, description, field_type, is_required, options, value)
SELECT id, 'project_param', project_id, template_id, name, description, field_type, is_required, options, value
FROM project_params;

INSERT INTO fields (id, kind, board_id, name, description, field_type, is_required, options)
SELECT id, 'board_field', board_id, name, description, field_type, is_required, options
FROM board_fields;

ALTER TABLE task_field_values DROP CONSTRAINT IF EXISTS task_field_values_field_id_fkey;
ALTER TABLE task_field_values ADD CONSTRAINT task_field_values_field_id_fkey
    FOREIGN KEY (field_id) REFERENCES fields(id) ON DELETE CASCADE;

DROP TABLE project_params;
DROP TABLE board_fields;
