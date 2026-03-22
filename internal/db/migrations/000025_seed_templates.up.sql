-- Seed default project templates
-- Encoding: UTF-8

SET client_encoding = 'UTF8';

-- ============================================================
-- 1. Шаблон «Scrum стандартный»
-- ============================================================

INSERT INTO project_templates (id, name, description, project_type)
SELECT uuid_generate_v4(),
       'Scrum стандартный',
       'Стандартный шаблон для Scrum-проектов с настройками по умолчанию',
       'scrum'
WHERE NOT EXISTS (
    SELECT 1 FROM project_templates WHERE name = 'Scrum стандартный' AND project_type = 'scrum'
);

-- Доска «Основная доска» для Scrum (только если у шаблона ещё нет досок)
INSERT INTO template_boards (id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by)
SELECT
    uuid_generate_v4(),
    pt.id,
    'Основная доска',
    'Доска для основного хода разработки',
    true,
    1,
    'priority',
    'story_points',
    ''
FROM project_templates pt
WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
  AND NOT EXISTS (SELECT 1 FROM template_boards WHERE template_id = pt.id);

-- Колонки Scrum-доски (только если у доски ещё нет колонок)
INSERT INTO template_board_columns (board_id, name, system_type, wip_limit, "order", is_locked)
SELECT tb.id, v.name, v.sys_type, NULL, v.ord, v.locked
FROM template_boards tb
JOIN project_templates pt ON tb.template_id = pt.id
CROSS JOIN (VALUES
    ('Бэклог спринта', 'initial', 1, true),
    ('В работе', 'in_progress', 2, false),
    ('На проверке', 'in_progress', 3, false),
    ('Выполнено', 'completed', 4, false)
) AS v(name, sys_type, ord, locked)
WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
  AND NOT EXISTS (SELECT 1 FROM template_board_columns WHERE board_id = tb.id);

-- Значения приоритетов Scrum-доски (только если ещё нет значений)
INSERT INTO template_board_priority_values (board_id, value, "order")
SELECT tb.id, v.value, v.ord
FROM template_boards tb
JOIN project_templates pt ON tb.template_id = pt.id
CROSS JOIN (VALUES
    ('Низкий', 1),
    ('Средний', 2),
    ('Высокий', 3),
    ('Критичный', 4)
) AS v(value, ord)
WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
  AND NOT EXISTS (SELECT 1 FROM template_board_priority_values WHERE board_id = tb.id);

-- ============================================================
-- 2. Шаблон «Kanban стандартный»
-- ============================================================

INSERT INTO project_templates (id, name, description, project_type)
SELECT uuid_generate_v4(),
       'Kanban стандартный',
       'Стандартный шаблон для Kanban-проектов с поддержкой WIP лимитов',
       'kanban'
WHERE NOT EXISTS (
    SELECT 1 FROM project_templates WHERE name = 'Kanban стандартный' AND project_type = 'kanban'
);

-- Доска «Основная доска» для Kanban (только если у шаблона ещё нет досок)
INSERT INTO template_boards (id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by)
SELECT
    uuid_generate_v4(),
    pt.id,
    'Основная доска',
    'Kanban-доска с поддержкой WIP лимитов',
    true,
    1,
    'service_class',
    'time',
    'service_class'
FROM project_templates pt
WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
  AND NOT EXISTS (SELECT 1 FROM template_boards WHERE template_id = pt.id);

-- Колонки Kanban-доски (только если у доски ещё нет колонок)
INSERT INTO template_board_columns (board_id, name, system_type, wip_limit, "order", is_locked)
SELECT tb.id, v.name, v.sys_type, NULL, v.ord, false
FROM template_boards tb
JOIN project_templates pt ON tb.template_id = pt.id
CROSS JOIN (VALUES
    ('Надо сделать', 'initial', 1),
    ('Готово к работе', 'initial', 2),
    ('В работе', 'in_progress', 3),
    ('На проверке', 'in_progress', 4),
    ('Выполнено', 'completed', 5)
) AS v(name, sys_type, ord)
WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
  AND NOT EXISTS (SELECT 1 FROM template_board_columns WHERE board_id = tb.id);

-- Дорожки Kanban-доски (только если ещё нет дорожек)
INSERT INTO template_board_swimlanes (board_id, name, value, wip_limit, "order")
SELECT tb.id, v.name, v.name, NULL, v.ord
FROM template_boards tb
JOIN project_templates pt ON tb.template_id = pt.id
CROSS JOIN (VALUES
    ('Ускоренный', 1),
    ('С фиксированной датой', 2),
    ('Стандартный', 3),
    ('Нематериальный', 4)
) AS v(name, ord)
WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
  AND NOT EXISTS (SELECT 1 FROM template_board_swimlanes WHERE board_id = tb.id);

-- Значения классов обслуживания Kanban-доски (только если ещё нет значений)
INSERT INTO template_board_priority_values (board_id, value, "order")
SELECT tb.id, v.value, v.ord
FROM template_boards tb
JOIN project_templates pt ON tb.template_id = pt.id
CROSS JOIN (VALUES
    ('Ускоренный', 1),
    ('С фиксированной датой', 2),
    ('Стандартный', 3),
    ('Нематериальный', 4)
) AS v(value, ord)
WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
  AND NOT EXISTS (SELECT 1 FROM template_board_priority_values WHERE board_id = tb.id);
