-- Migration: Re-apply default column names in proper UTF-8 encoding.
-- Overwrites column names inserted by migration 000009 which may have had encoding issues.

-- Scrum template boards:
UPDATE columns c
SET name = col.name
FROM boards b
JOIN templates t ON t.id = b.template_id
CROSS JOIN (VALUES
    (1, E'\u0411\u044d\u043a\u043b\u043e\u0433 \u0441\u043f\u0440\u0438\u043d\u0442\u0430'),
    (2, E'\u0412 \u0440\u0430\u0431\u043e\u0442\u0435'),
    (3, E'\u041d\u0430 \u043f\u0440\u043e\u0432\u0435\u0440\u043a\u0435'),
    (4, E'\u0412\u044b\u043f\u043e\u043b\u043d\u0435\u043d\u043e')
) AS col(sort_order, name)
WHERE c.board_id = b.id
  AND t.project_type = 'scrum'
  AND c.sort_order = col.sort_order;

-- Kanban template boards:
UPDATE columns c
SET name = col.name
FROM boards b
JOIN templates t ON t.id = b.template_id
CROSS JOIN (VALUES
    (1, E'\u041d\u0430\u0434\u043e \u0441\u0434\u0435\u043b\u0430\u0442\u044c'),
    (2, E'\u0413\u043e\u0442\u043e\u0432\u043e \u043a \u0440\u0430\u0431\u043e\u0442\u0435'),
    (3, E'\u0412 \u0440\u0430\u0431\u043e\u0442\u0435'),
    (4, E'\u041d\u0430 \u043f\u0440\u043e\u0432\u0435\u0440\u043a\u0435'),
    (5, E'\u0412\u044b\u043f\u043e\u043b\u043d\u0435\u043d\u043e')
) AS col(sort_order, name)
WHERE c.board_id = b.id
  AND t.project_type = 'kanban'
  AND c.sort_order = col.sort_order;

-- Scrum project boards:
UPDATE columns c
SET name = col.name
FROM boards b
JOIN projects p ON p.id = b.project_id
CROSS JOIN (VALUES
    (1, E'\u0411\u044d\u043a\u043b\u043e\u0433 \u0441\u043f\u0440\u0438\u043d\u0442\u0430'),
    (2, E'\u0412 \u0440\u0430\u0431\u043e\u0442\u0435'),
    (3, E'\u041d\u0430 \u043f\u0440\u043e\u0432\u0435\u0440\u043a\u0435'),
    (4, E'\u0412\u044b\u043f\u043e\u043b\u043d\u0435\u043d\u043e')
) AS col(sort_order, name)
WHERE c.board_id = b.id
  AND p.project_type = 'scrum'
  AND c.sort_order = col.sort_order;

-- Kanban project boards:
UPDATE columns c
SET name = col.name
FROM boards b
JOIN projects p ON p.id = b.project_id
CROSS JOIN (VALUES
    (1, E'\u041d\u0430\u0434\u043e \u0441\u0434\u0435\u043b\u0430\u0442\u044c'),
    (2, E'\u0413\u043e\u0442\u043e\u0432\u043e \u043a \u0440\u0430\u0431\u043e\u0442\u0435'),
    (3, E'\u0412 \u0440\u0430\u0431\u043e\u0442\u0435'),
    (4, E'\u041d\u0430 \u043f\u0440\u043e\u0432\u0435\u0440\u043a\u0435'),
    (5, E'\u0412\u044b\u043f\u043e\u043b\u043d\u0435\u043d\u043e')
) AS col(sort_order, name)
WHERE c.board_id = b.id
  AND p.project_type = 'kanban'
  AND c.sort_order = col.sort_order;
