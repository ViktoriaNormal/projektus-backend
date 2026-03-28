-- Откат 000004

ALTER TABLE roles ADD COLUMN sort_order INT DEFAULT 0 NOT NULL;

-- Восстановить проектные права без .manage
UPDATE role_permissions SET permission_code = 'project.sprints'          WHERE permission_code = 'project.sprints.manage';
UPDATE role_permissions SET permission_code = 'project.boards'            WHERE permission_code = 'project.boards.manage';
UPDATE role_permissions SET permission_code = 'project.analytics'         WHERE permission_code = 'project.analytics.manage';
UPDATE role_permissions SET permission_code = 'project.backlog'           WHERE permission_code = 'project.backlog.manage';
UPDATE role_permissions SET permission_code = 'project.tasks'             WHERE permission_code = 'project.tasks.manage';
UPDATE role_permissions SET permission_code = 'project.project_settings'  WHERE permission_code = 'project.project_settings.manage';
UPDATE role_permissions SET permission_code = 'project.wip_limits'        WHERE permission_code = 'project.wip_limits.manage';

-- Восстановить разделённые системные права
DELETE FROM role_permissions WHERE permission_code = 'system.projects.manage';

INSERT INTO role_permissions (role_id, permission_code)
SELECT r.id, code
FROM roles r, (VALUES
    ('system.projects.create'),
    ('system.projects.view_all'),
    ('system.projects.edit_all'),
    ('system.projects.delete'),
    ('system.projects.archive')
) AS t(code)
WHERE r.scope = 'system' AND r.is_admin = true
ON CONFLICT DO NOTHING;
