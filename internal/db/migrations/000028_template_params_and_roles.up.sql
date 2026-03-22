-- Template project params and roles
-- Encoding: UTF-8

SET client_encoding = 'UTF8';

-- Справочник: системные параметры проектов
CREATE TABLE IF NOT EXISTS ref_system_project_params (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    options JSONB,
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO ref_system_project_params (key, name, field_type, is_required, options, sort_order) VALUES
    ('name', 'Название', 'text', true, NULL, 1),
    ('owner', 'Ответственный за проект', 'user', true, NULL, 2),
    ('description', 'Описание', 'text', false, NULL, 3),
    ('status', 'Статус', 'select', true, '["Активный", "Архивирован", "Приостановлен"]'::jsonb, 4)
ON CONFLICT (key) DO NOTHING;

-- Справочник: области прав доступа
CREATE TABLE IF NOT EXISTS ref_permission_areas (
    area TEXT NOT NULL,
    project_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (area, project_type)
);

INSERT INTO ref_permission_areas (area, project_type, name, description, sort_order) VALUES
    ('sprints', 'scrum', 'Управление спринтами', 'Просмотр, создание, запуск, завершение и удаление спринтов', 1),
    ('boards', 'scrum', 'Управление досками', 'Управление досками, колонками, дорожками и их параметрами', 2),
    ('analytics', 'scrum', 'Аналитика', 'Управление Scrum-отчётностью', 3),
    ('backlog', 'scrum', 'Управление бэклогами', 'Добавление задач в бэклог продукта, приоритизация и перенос в спринты', 4),
    ('tasks', 'scrum', 'Управление задачами', 'Просмотр, создание, редактирование, перемещение, удаление и комментирование задач', 5),
    ('project_settings', 'scrum', 'Параметры проекта', 'Управление параметрами проектов (включая кастомные), ролями участников и участниками', 6),
    ('boards', 'kanban', 'Управление досками', 'Управление досками, колонками, дорожками и их параметрами', 1),
    ('wip_limits', 'kanban', 'WIP-лимиты', 'Управление WIP-лимитами (ограничениями незавершённой работы) для колонок и дорожек', 2),
    ('analytics', 'kanban', 'Аналитика', 'Управление Kanban-отчётностью и выполнение прогнозирования сроков завершения работ методом Монте-Карло', 3),
    ('tasks', 'kanban', 'Управление задачами', 'Просмотр, создание, редактирование, перемещение, удаление и комментирование задач', 4),
    ('project_settings', 'kanban', 'Параметры проекта', 'Управление параметрами проектов (включая кастомные), ролями участников и участниками', 5)
ON CONFLICT (area, project_type) DO NOTHING;

-- Справочник: уровни доступа
CREATE TABLE IF NOT EXISTS ref_access_levels (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO ref_access_levels (key, name, sort_order) VALUES
    ('full', 'Полный доступ', 1),
    ('view', 'Только просмотр', 2),
    ('none', 'Нет доступа', 3)
ON CONFLICT (key) DO NOTHING;

-- Кастомные параметры проекта в шаблоне
CREATE TABLE IF NOT EXISTS template_project_params (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES project_templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    "order" INT NOT NULL DEFAULT 1,
    options JSONB
);

-- Роли в шаблоне
CREATE TABLE IF NOT EXISTS template_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES project_templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    "order" INT NOT NULL DEFAULT 1
);

-- Права роли в шаблоне
CREATE TABLE IF NOT EXISTS template_role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id UUID NOT NULL REFERENCES template_roles(id) ON DELETE CASCADE,
    area TEXT NOT NULL,
    access TEXT NOT NULL DEFAULT 'none',
    UNIQUE(role_id, area)
);

-- Seed default roles for existing Scrum template
INSERT INTO template_roles (template_id, name, description, is_default, "order")
SELECT pt.id, 'Владелец продукта', 'Полный доступ ко всем сущностям проекта', true, 1
FROM project_templates pt WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
AND NOT EXISTS (SELECT 1 FROM template_roles tr WHERE tr.template_id = pt.id AND tr.name = 'Владелец продукта');

INSERT INTO template_role_permissions (role_id, area, access)
SELECT tr.id, v.area, v.access
FROM template_roles tr
JOIN project_templates pt ON tr.template_id = pt.id
CROSS JOIN (VALUES ('sprints','full'),('boards','full'),('analytics','full'),('backlog','full'),('tasks','full'),('project_settings','full')) AS v(area, access)
WHERE pt.name = 'Scrum стандартный' AND tr.name = 'Владелец продукта'
AND NOT EXISTS (SELECT 1 FROM template_role_permissions trp WHERE trp.role_id = tr.id);

INSERT INTO template_roles (template_id, name, description, is_default, "order")
SELECT pt.id, 'Scrum-мастер', 'Управление процессом, спринтами и досками', true, 2
FROM project_templates pt WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
AND NOT EXISTS (SELECT 1 FROM template_roles tr WHERE tr.template_id = pt.id AND tr.name = 'Scrum-мастер');

INSERT INTO template_role_permissions (role_id, area, access)
SELECT tr.id, v.area, v.access
FROM template_roles tr
JOIN project_templates pt ON tr.template_id = pt.id
CROSS JOIN (VALUES ('sprints','full'),('boards','full'),('analytics','full'),('backlog','view'),('tasks','view'),('project_settings','view')) AS v(area, access)
WHERE pt.name = 'Scrum стандартный' AND tr.name = 'Scrum-мастер'
AND NOT EXISTS (SELECT 1 FROM template_role_permissions trp WHERE trp.role_id = tr.id);

INSERT INTO template_roles (template_id, name, description, is_default, "order")
SELECT pt.id, 'Член команды разработки', 'Работа с задачами, досками, спринтами и бэклогами', true, 3
FROM project_templates pt WHERE pt.name = 'Scrum стандартный' AND pt.project_type = 'scrum'
AND NOT EXISTS (SELECT 1 FROM template_roles tr WHERE tr.template_id = pt.id AND tr.name = 'Член команды разработки');

INSERT INTO template_role_permissions (role_id, area, access)
SELECT tr.id, v.area, v.access
FROM template_roles tr
JOIN project_templates pt ON tr.template_id = pt.id
CROSS JOIN (VALUES ('sprints','full'),('boards','full'),('analytics','none'),('backlog','full'),('tasks','full'),('project_settings','view')) AS v(area, access)
WHERE pt.name = 'Scrum стандартный' AND tr.name = 'Член команды разработки'
AND NOT EXISTS (SELECT 1 FROM template_role_permissions trp WHERE trp.role_id = tr.id);

-- Seed default roles for existing Kanban template
INSERT INTO template_roles (template_id, name, description, is_default, "order")
SELECT pt.id, 'Менеджер проекта', 'Полный доступ ко всем сущностям проекта, включая WIP-лимиты', true, 1
FROM project_templates pt WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
AND NOT EXISTS (SELECT 1 FROM template_roles tr WHERE tr.template_id = pt.id AND tr.name = 'Менеджер проекта');

INSERT INTO template_role_permissions (role_id, area, access)
SELECT tr.id, v.area, v.access
FROM template_roles tr
JOIN project_templates pt ON tr.template_id = pt.id
CROSS JOIN (VALUES ('boards','full'),('wip_limits','full'),('analytics','full'),('tasks','full'),('project_settings','full')) AS v(area, access)
WHERE pt.name = 'Kanban стандартный' AND tr.name = 'Менеджер проекта'
AND NOT EXISTS (SELECT 1 FROM template_role_permissions trp WHERE trp.role_id = tr.id);

INSERT INTO template_roles (template_id, name, description, is_default, "order")
SELECT pt.id, 'Базовый участник проекта', 'Работа с задачами и досками', true, 2
FROM project_templates pt WHERE pt.name = 'Kanban стандартный' AND pt.project_type = 'kanban'
AND NOT EXISTS (SELECT 1 FROM template_roles tr WHERE tr.template_id = pt.id AND tr.name = 'Базовый участник проекта');

INSERT INTO template_role_permissions (role_id, area, access)
SELECT tr.id, v.area, v.access
FROM template_roles tr
JOIN project_templates pt ON tr.template_id = pt.id
CROSS JOIN (VALUES ('boards','full'),('wip_limits','view'),('analytics','none'),('tasks','full'),('project_settings','view')) AS v(area, access)
WHERE pt.name = 'Kanban стандартный' AND tr.name = 'Базовый участник проекта'
AND NOT EXISTS (SELECT 1 FROM template_role_permissions trp WHERE trp.role_id = tr.id);
