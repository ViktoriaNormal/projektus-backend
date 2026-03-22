-- Add note field to template board columns and swimlanes

ALTER TABLE template_board_columns ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';
ALTER TABLE template_board_swimlanes ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';
