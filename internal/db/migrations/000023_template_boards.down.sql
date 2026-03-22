DROP TABLE IF EXISTS template_board_fields;
DROP TABLE IF EXISTS template_board_priority_values;
DROP TABLE IF EXISTS template_board_swimlanes;
DROP TABLE IF EXISTS template_board_columns;
DROP TABLE IF EXISTS template_boards;
ALTER TABLE project_templates DROP COLUMN IF EXISTS updated_at;
