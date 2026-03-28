-- =============================================================================
-- 000004: Удалить sort_order из roles, обновить коды прав доступа
-- =============================================================================

-- ======================== 1. Удалить sort_order из roles ========================

ALTER TABLE roles DROP COLUMN sort_order;

-- ======================== 2. Обновить системные права ========================
-- Было: system.projects.create, system.projects.view_all, system.projects.edit_all,
--        system.projects.delete, system.projects.archive
-- Стало: system.projects.manage (единое право)

-- Удалить старые системные project-права
DELETE FROM role_permissions
WHERE permission_code IN (
    'system.projects.create',
    'system.projects.view_all',
    'system.projects.edit_all',
    'system.projects.delete',
    'system.projects.archive'
);

-- Добавить единое system.projects.manage с access='full'
INSERT INTO role_permissions (role_id, permission_code, access)
SELECT r.id, 'system.projects.manage', 'full'
FROM roles r
WHERE r.scope = 'system' AND r.is_admin = true
ON CONFLICT DO NOTHING;

-- Установить access='full' для всех системных прав, где access NULL
UPDATE role_permissions
SET access = 'full'
WHERE role_id IN (SELECT id FROM roles WHERE scope = 'system')
  AND access IS NULL;

-- ======================== 3. Обновить проектные/шаблонные права: добавить .manage ========================

UPDATE role_permissions SET permission_code = 'project.sprints.manage'          WHERE permission_code = 'project.sprints';
UPDATE role_permissions SET permission_code = 'project.boards.manage'            WHERE permission_code = 'project.boards';
UPDATE role_permissions SET permission_code = 'project.analytics.manage'         WHERE permission_code = 'project.analytics';
UPDATE role_permissions SET permission_code = 'project.backlog.manage'           WHERE permission_code = 'project.backlog';
UPDATE role_permissions SET permission_code = 'project.tasks.manage'             WHERE permission_code = 'project.tasks';
UPDATE role_permissions SET permission_code = 'project.project_settings.manage'  WHERE permission_code = 'project.project_settings';
UPDATE role_permissions SET permission_code = 'project.wip_limits.manage'        WHERE permission_code = 'project.wip_limits';
