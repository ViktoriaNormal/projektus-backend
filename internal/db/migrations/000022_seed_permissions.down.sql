-- Remove seeded permissions (cascade will clean role_permissions)
DELETE FROM permissions WHERE code IN (
    'system.users.manage',
    'system.password_policy.manage',
    'system.project_templates.manage',
    'system.projects.view_all',
    'system.projects.create',
    'system.projects.delete',
    'system.projects.archive',
    'system.projects.edit_all'
);
