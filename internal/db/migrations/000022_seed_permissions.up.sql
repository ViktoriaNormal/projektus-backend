-- Seed additional system permissions

INSERT INTO permissions (code, description) VALUES
    ('system.users.manage', 'Управление пользователями'),
    ('system.password_policy.manage', 'Управление парольной политикой'),
    ('system.project_templates.manage', 'Управление шаблонами проектов'),
    ('system.projects.view_all', 'Просмотр всех проектов'),
    ('system.projects.create', 'Создание новых проектов'),
    ('system.projects.delete', 'Удаление любых проектов'),
    ('system.projects.archive', 'Архивация/разархивация любых проектов'),
    ('system.projects.edit_all', 'Редактирование любых проектов')
ON CONFLICT (code) DO NOTHING;

-- Grant all new permissions to the Administrator role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Админ' AND r.scope = 'system' AND r.project_id IS NULL
  AND p.code IN (
    'system.users.manage',
    'system.password_policy.manage',
    'system.project_templates.manage',
    'system.projects.view_all',
    'system.projects.create',
    'system.projects.delete',
    'system.projects.archive',
    'system.projects.edit_all'
  )
ON CONFLICT DO NOTHING;
