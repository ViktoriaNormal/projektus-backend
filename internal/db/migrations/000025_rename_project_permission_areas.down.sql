UPDATE role_permissions SET permission_code = 'project.boards.manage' WHERE permission_code = 'project.boards';
UPDATE role_permissions SET permission_code = 'project.tasks.manage' WHERE permission_code = 'project.tasks';
UPDATE role_permissions SET permission_code = 'project.project_settings.manage' WHERE permission_code = 'project.settings';
UPDATE role_permissions SET permission_code = 'project.sprints.manage' WHERE permission_code = 'project.sprints';
UPDATE role_permissions SET permission_code = 'project.analytics.manage' WHERE permission_code = 'project.analytics';
