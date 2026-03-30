-- Migration: Reset all board columns to defaults for all templates and projects.
-- This fixes column order chaos caused by missing recompact logic on insert/delete.

-- Step 1: Move tasks to the first remaining column of their board (to avoid FK violations).
-- We'll reassign all tasks to a temporary NULL column_id, then delete columns, re-insert defaults,
-- and reassign tasks to the first "initial" column of their board.

-- Step 1a: Store task->board mapping before deleting columns.
CREATE TEMP TABLE task_board_map AS
SELECT t.id AS task_id, c.board_id
FROM tasks t
JOIN columns c ON c.id = t.column_id;

-- Step 1b: Temporarily drop the NOT NULL constraint on tasks.column_id so we can delete columns.
ALTER TABLE tasks ALTER COLUMN column_id DROP NOT NULL;

-- Step 1c: Set column_id to NULL for all tasks (we'll reassign after).
UPDATE tasks SET column_id = NULL;

-- Step 1d: Also set swimlane_id to NULL for tasks (swimlanes don't change, but column reassignment is needed).
-- No action needed for swimlanes.

-- Step 2: Delete ALL columns from ALL boards.
DELETE FROM columns;

-- Step 3: Re-insert default columns for template boards (based on template project_type).
-- Scrum templates:
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked, note)
SELECT b.id, col.name, col.system_type, NULL, col.sort_order, col.is_locked, ''
FROM boards b
JOIN templates t ON t.id = b.template_id
CROSS JOIN (VALUES
    ('Бэклог спринта', 'initial', 1, true),
    ('В работе', 'in_progress', 2, false),
    ('На проверке', 'in_progress', 3, false),
    ('Выполнено', 'completed', 4, false)
) AS col(name, system_type, sort_order, is_locked)
WHERE t.project_type = 'scrum';

-- Kanban templates:
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked, note)
SELECT b.id, col.name, col.system_type, NULL, col.sort_order, col.is_locked, ''
FROM boards b
JOIN templates t ON t.id = b.template_id
CROSS JOIN (VALUES
    ('Надо сделать', 'initial', 1, false),
    ('Готово к работе', 'initial', 2, false),
    ('В работе', 'in_progress', 3, false),
    ('На проверке', 'in_progress', 4, false),
    ('Выполнено', 'completed', 5, false)
) AS col(name, system_type, sort_order, is_locked)
WHERE t.project_type = 'kanban';

-- Step 4: Re-insert default columns for project boards (based on project project_type).
-- Scrum projects:
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked, note)
SELECT b.id, col.name, col.system_type, NULL, col.sort_order, col.is_locked, ''
FROM boards b
JOIN projects p ON p.id = b.project_id
CROSS JOIN (VALUES
    ('Бэклог спринта', 'initial', 1, true),
    ('В работе', 'in_progress', 2, false),
    ('На проверке', 'in_progress', 3, false),
    ('Выполнено', 'completed', 4, false)
) AS col(name, system_type, sort_order, is_locked)
WHERE p.project_type = 'scrum';

-- Kanban projects:
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked, note)
SELECT b.id, col.name, col.system_type, NULL, col.sort_order, col.is_locked, ''
FROM boards b
JOIN projects p ON p.id = b.project_id
CROSS JOIN (VALUES
    ('Надо сделать', 'initial', 1, false),
    ('Готово к работе', 'initial', 2, false),
    ('В работе', 'in_progress', 3, false),
    ('На проверке', 'in_progress', 4, false),
    ('Выполнено', 'completed', 5, false)
) AS col(name, system_type, sort_order, is_locked)
WHERE p.project_type = 'kanban';

-- Step 5: Reassign tasks to the first "initial" column of their original board.
UPDATE tasks t
SET column_id = (
    SELECT c.id FROM columns c
    WHERE c.board_id = tbm.board_id AND c.system_type = 'initial'
    ORDER BY c.sort_order ASC
    LIMIT 1
)
FROM task_board_map tbm
WHERE t.id = tbm.task_id;

-- Step 6: Restore NOT NULL constraint.
ALTER TABLE tasks ALTER COLUMN column_id SET NOT NULL;

-- Step 7: Cleanup.
DROP TABLE task_board_map;
