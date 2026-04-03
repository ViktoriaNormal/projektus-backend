-- Rename project permission area codes to shorter names.
UPDATE role_permissions SET permission_code = 'project.boards' WHERE permission_code = 'project.boards.manage';
UPDATE role_permissions SET permission_code = 'project.tasks' WHERE permission_code = 'project.tasks.manage';
UPDATE role_permissions SET permission_code = 'project.settings' WHERE permission_code = 'project.project_settings.manage';
UPDATE role_permissions SET permission_code = 'project.sprints' WHERE permission_code = 'project.sprints.manage';
UPDATE role_permissions SET permission_code = 'project.analytics' WHERE permission_code = 'project.analytics.manage';
-- Remove deprecated areas (backlog, wip_limits) — these are now covered by project.boards and project.tasks.
DELETE FROM role_permissions WHERE permission_code IN ('project.backlog.manage', 'project.wip_limits.manage');
