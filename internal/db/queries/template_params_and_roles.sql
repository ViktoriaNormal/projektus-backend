-- Template project params

-- name: ListTemplateProjectParams :many
SELECT id, template_id, name, field_type, is_required, "order", options
FROM template_project_params
WHERE template_id = $1
ORDER BY "order" ASC;

-- name: GetTemplateProjectParamByID :one
SELECT id, template_id, name, field_type, is_required, "order", options
FROM template_project_params
WHERE id = $1;

-- name: CreateTemplateProjectParam :one
INSERT INTO template_project_params (template_id, name, field_type, is_required, "order", options)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, template_id, name, field_type, is_required, "order", options;

-- name: UpdateTemplateProjectParam :one
UPDATE template_project_params
SET name = $2, is_required = $3, options = $4
WHERE id = $1
RETURNING id, template_id, name, field_type, is_required, "order", options;

-- name: DeleteTemplateProjectParamByID :exec
DELETE FROM template_project_params WHERE id = $1;

-- name: UpdateTemplateProjectParamOrder :exec
UPDATE template_project_params SET "order" = $2 WHERE id = $1;

-- Template roles

-- name: ListTemplateRoles :many
SELECT id, template_id, name, description, is_default, "order"
FROM template_roles
WHERE template_id = $1
ORDER BY "order" ASC;

-- name: GetTemplateRoleByID :one
SELECT id, template_id, name, description, is_default, "order"
FROM template_roles
WHERE id = $1;

-- name: CreateTemplateRole :one
INSERT INTO template_roles (template_id, name, description, is_default, "order")
VALUES ($1, $2, $3, $4, $5)
RETURNING id, template_id, name, description, is_default, "order";

-- name: UpdateTemplateRole :one
UPDATE template_roles
SET name = $2, description = $3
WHERE id = $1
RETURNING id, template_id, name, description, is_default, "order";

-- name: DeleteTemplateRoleByID :exec
DELETE FROM template_roles WHERE id = $1;

-- name: UpdateTemplateRoleOrder :exec
UPDATE template_roles SET "order" = $2 WHERE id = $1;

-- name: CountTemplateRolesByTemplateID :one
SELECT COUNT(*)::int AS count FROM template_roles WHERE template_id = $1;

-- Template role permissions

-- name: ListTemplateRolePermissions :many
SELECT id, role_id, area, access
FROM template_role_permissions
WHERE role_id = $1;

-- name: UpsertTemplateRolePermission :exec
INSERT INTO template_role_permissions (role_id, area, access)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, area) DO UPDATE SET access = EXCLUDED.access;

-- name: DeleteTemplateRolePermissionsByRoleID :exec
DELETE FROM template_role_permissions WHERE role_id = $1;

-- Reference queries for new tables

-- name: ListRefSystemProjectParams :many
SELECT key, name, field_type, is_required, options, sort_order
FROM ref_system_project_params
ORDER BY sort_order ASC;

-- name: ListRefPermissionAreas :many
SELECT area, project_type, name, description, sort_order
FROM ref_permission_areas
ORDER BY project_type, sort_order ASC;

-- name: ListRefAccessLevels :many
SELECT key, name, sort_order
FROM ref_access_levels
ORDER BY sort_order ASC;
