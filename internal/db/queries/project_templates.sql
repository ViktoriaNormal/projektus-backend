-- Project templates

-- name: ListProjectTemplates :many
SELECT pt.id, pt.name, pt.description, pt.project_type, pt.created_at, pt.updated_at,
       (SELECT COUNT(*) FROM template_boards tb WHERE tb.template_id = pt.id)::int AS board_count
FROM project_templates pt
ORDER BY pt.project_type ASC;

-- name: GetProjectTemplateByID :one
SELECT id, name, description, project_type, created_at, updated_at
FROM project_templates
WHERE id = $1;

-- name: CreateProjectTemplate :one
INSERT INTO project_templates (name, description, project_type)
VALUES ($1, $2, $3)
RETURNING id, name, description, project_type, created_at, updated_at;

-- name: UpdateProjectTemplate :one
UPDATE project_templates
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, project_type, created_at, updated_at;

-- name: DeleteProjectTemplate :exec
DELETE FROM project_templates WHERE id = $1;

-- name: IsTemplateInUse :one
SELECT EXISTS(SELECT 1 FROM projects WHERE project_type = (SELECT project_type FROM project_templates pt2 WHERE pt2.id = $1) LIMIT 0) AS in_use;

-- Template boards

-- name: ListTemplateBoardsByTemplateID :many
SELECT id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by
FROM template_boards
WHERE template_id = $1
ORDER BY "order" ASC;

-- name: GetTemplateBoardByID :one
SELECT id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by
FROM template_boards
WHERE id = $1;

-- name: CreateTemplateBoard :one
INSERT INTO template_boards (template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by;

-- name: UpdateTemplateBoard :one
UPDATE template_boards
SET name = $2, description = $3, is_default = $4, "order" = $5, priority_type = $6, estimation_unit = $7, swimlane_group_by = $8
WHERE id = $1
RETURNING id, template_id, name, description, is_default, "order", priority_type, estimation_unit, swimlane_group_by;

-- name: DeleteTemplateBoardByID :exec
DELETE FROM template_boards WHERE id = $1;

-- name: DeleteTemplateBoardsByTemplateID :exec
DELETE FROM template_boards WHERE template_id = $1;

-- name: CountTemplateBoardsByTemplateID :one
SELECT COUNT(*)::int AS count FROM template_boards WHERE template_id = $1;

-- name: UnsetDefaultBoardByTemplateID :exec
UPDATE template_boards SET is_default = false WHERE template_id = $1 AND is_default = true;

-- name: UpdateTemplateBoardOrder :exec
UPDATE template_boards SET "order" = $2 WHERE id = $1;

-- Template board columns

-- name: ListTemplateBoardColumns :many
SELECT id, board_id, name, system_type, wip_limit, "order", is_locked, note
FROM template_board_columns
WHERE board_id = $1
ORDER BY "order" ASC;

-- name: GetTemplateBoardColumnByID :one
SELECT id, board_id, name, system_type, wip_limit, "order", is_locked, note
FROM template_board_columns
WHERE id = $1;

-- name: CreateTemplateBoardColumn :one
INSERT INTO template_board_columns (board_id, name, system_type, wip_limit, "order", is_locked, note)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, board_id, name, system_type, wip_limit, "order", is_locked, note;

-- name: UpdateTemplateBoardColumn :one
UPDATE template_board_columns
SET name = $2, system_type = $3, wip_limit = $4, note = $5
WHERE id = $1
RETURNING id, board_id, name, system_type, wip_limit, "order", is_locked, note;

-- name: DeleteTemplateBoardColumnByID :exec
DELETE FROM template_board_columns WHERE id = $1;

-- name: DeleteTemplateBoardColumnsByBoardID :exec
DELETE FROM template_board_columns WHERE board_id = $1;

-- name: UpdateTemplateBoardColumnOrder :exec
UPDATE template_board_columns SET "order" = $2 WHERE id = $1;

-- Template board swimlanes

-- name: ListTemplateBoardSwimlanes :many
SELECT id, board_id, name, value, wip_limit, "order", note
FROM template_board_swimlanes
WHERE board_id = $1
ORDER BY "order" ASC;

-- name: GetTemplateBoardSwimlaneByID :one
SELECT id, board_id, name, value, wip_limit, "order", note
FROM template_board_swimlanes
WHERE id = $1;

-- name: CreateTemplateBoardSwimlane :one
INSERT INTO template_board_swimlanes (board_id, name, value, wip_limit, "order", note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, board_id, name, value, wip_limit, "order", note;

-- name: UpdateTemplateBoardSwimlane :one
UPDATE template_board_swimlanes
SET wip_limit = $2, note = $3
WHERE id = $1
RETURNING id, board_id, name, value, wip_limit, "order", note;

-- name: DeleteTemplateBoardSwimlaneByID :exec
DELETE FROM template_board_swimlanes WHERE id = $1;

-- name: DeleteTemplateBoardSwimlanesByBoardID :exec
DELETE FROM template_board_swimlanes WHERE board_id = $1;

-- name: UpdateTemplateBoardSwimlaneOrder :exec
UPDATE template_board_swimlanes SET "order" = $2 WHERE id = $1;

-- Template board priority values

-- name: ListTemplateBoardPriorityValues :many
SELECT id, board_id, value, "order"
FROM template_board_priority_values
WHERE board_id = $1
ORDER BY "order" ASC;

-- name: CreateTemplateBoardPriorityValue :one
INSERT INTO template_board_priority_values (board_id, value, "order")
VALUES ($1, $2, $3)
RETURNING id, board_id, value, "order";

-- name: DeleteTemplateBoardPriorityValuesByBoardID :exec
DELETE FROM template_board_priority_values WHERE board_id = $1;

-- Template board custom fields

-- name: ListTemplateBoardFields :many
SELECT id, board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config
FROM template_board_fields
WHERE board_id = $1
ORDER BY "order" ASC;

-- name: ListTemplateBoardCustomFields :many
SELECT id, board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config
FROM template_board_fields
WHERE board_id = $1 AND is_system = false
ORDER BY "order" ASC;

-- name: GetTemplateBoardFieldByID :one
SELECT id, board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config
FROM template_board_fields
WHERE id = $1;

-- name: CreateTemplateBoardField :one
INSERT INTO template_board_fields (board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config;

-- name: UpdateTemplateBoardField :one
UPDATE template_board_fields
SET name = $2, is_required = $3, options = $4
WHERE id = $1
RETURNING id, board_id, code, name, field_type, is_system, is_required, is_active, "order", options, config;

-- name: DeleteTemplateBoardFieldByID :exec
DELETE FROM template_board_fields WHERE id = $1;

-- name: DeleteTemplateBoardFieldsByBoardID :exec
DELETE FROM template_board_fields WHERE board_id = $1;

-- name: DeleteNonSystemFieldsByBoardID :exec
DELETE FROM template_board_fields WHERE board_id = $1 AND is_system = false;

-- name: UpdateTemplateBoardFieldOrder :exec
UPDATE template_board_fields SET "order" = $2 WHERE id = $1;
