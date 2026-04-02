-- Template project params

-- name: ListTemplateProjectParams :many
SELECT id, template_id, name, field_type, is_required, options
FROM project_params
WHERE template_id = $1;

-- name: GetTemplateProjectParamByID :one
SELECT id, template_id, name, field_type, is_required, options
FROM project_params
WHERE id = $1;

-- name: CreateTemplateProjectParam :one
INSERT INTO project_params (template_id, name, field_type, is_required, options)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, template_id, name, field_type, is_required, options;

-- name: UpdateTemplateProjectParam :one
UPDATE project_params
SET name = $2, is_required = $3, options = $4
WHERE id = $1
RETURNING id, template_id, name, field_type, is_required, options;

-- name: DeleteTemplateProjectParamByID :exec
DELETE FROM project_params WHERE id = $1;

-- Template roles (in unified roles table, scope='template')

-- name: ListTemplateRoles :many
SELECT id, template_id, name, is_admin
FROM roles
WHERE template_id = $1;

-- name: GetTemplateRoleByID :one
SELECT id, template_id, name, is_admin
FROM roles
WHERE id = $1;

-- name: CreateTemplateRole :one
INSERT INTO roles (template_id, scope, name, description)
VALUES ($1, 'template', $2, $3)
RETURNING id, template_id, name, is_admin;

-- name: UpdateTemplateRole :one
UPDATE roles
SET name = $2, description = $3
WHERE id = $1
RETURNING id, template_id, name, is_admin;

-- name: DeleteTemplateRoleByID :exec
DELETE FROM roles WHERE id = $1;

-- name: CountTemplateRolesByTemplateID :one
SELECT COUNT(*)::int AS count FROM roles WHERE template_id = $1;

-- Template role permissions (uses permission_code directly)

-- name: ListTemplateRolePermissions :many
SELECT rp.role_id, rp.permission_code, rp.access
FROM role_permissions rp
WHERE rp.role_id = $1;

-- name: UpsertTemplateRolePermission :exec
INSERT INTO role_permissions (role_id, permission_code, access)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, permission_code) DO UPDATE SET access = EXCLUDED.access;

-- name: DeleteTemplateRolePermissionsByRoleID :exec
DELETE FROM role_permissions WHERE role_id = $1;
