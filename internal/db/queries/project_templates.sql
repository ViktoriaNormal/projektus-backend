-- Project templates

-- name: ListProjectTemplates :many
SELECT t.id, t.name, t.description, t.project_type,
       (SELECT COUNT(*) FROM boards b WHERE b.template_id = t.id)::int AS board_count
FROM templates t
ORDER BY t.project_type ASC;

-- name: GetProjectTemplateByID :one
SELECT id, name, description, project_type
FROM templates
WHERE id = $1;

-- name: CreateProjectTemplate :one
INSERT INTO templates (name, description, project_type)
VALUES ($1, $2, $3)
RETURNING id, name, description, project_type;

-- name: UpdateProjectTemplate :one
UPDATE templates
SET name = $2, description = $3
WHERE id = $1
RETURNING id, name, description, project_type;

-- name: DeleteProjectTemplate :exec
DELETE FROM templates WHERE id = $1;

-- name: GetProjectTemplateByType :one
SELECT id, name, description, project_type
FROM templates
WHERE project_type = $1
LIMIT 1;

-- name: IsTemplateInUse :one
SELECT EXISTS(SELECT 1 FROM projects WHERE project_type = (SELECT project_type FROM templates t2 WHERE t2.id = $1) LIMIT 0) AS in_use;

-- Template boards (now in unified boards table, filtered by template_id)

-- name: ListTemplateBoardsByTemplateID :many
SELECT id, template_id, name, description, is_default, sort_order, priority_type, estimation_unit, swimlane_group_by
FROM boards
WHERE template_id = $1
ORDER BY sort_order ASC;

-- name: GetTemplateBoardByID :one
SELECT id, template_id, name, description, is_default, sort_order, priority_type, estimation_unit, swimlane_group_by
FROM boards
WHERE id = $1;

-- name: CreateTemplateBoard :one
INSERT INTO boards (template_id, name, description, is_default, sort_order, priority_type, estimation_unit, swimlane_group_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, template_id, name, description, is_default, sort_order, priority_type, estimation_unit, swimlane_group_by;

-- name: UpdateTemplateBoard :one
UPDATE boards
SET name = $2, description = $3, is_default = $4, sort_order = $5, priority_type = $6, estimation_unit = $7, swimlane_group_by = $8
WHERE id = $1
RETURNING id, template_id, name, description, is_default, sort_order, priority_type, estimation_unit, swimlane_group_by;

-- name: DeleteTemplateBoardByID :exec
DELETE FROM boards WHERE id = $1;

-- name: CountTemplateBoardsByTemplateID :one
SELECT COUNT(*)::int AS count FROM boards WHERE template_id = $1;

-- name: UnsetDefaultBoardByTemplateID :exec
UPDATE boards SET is_default = false WHERE template_id = $1 AND is_default = true;

-- name: UpdateTemplateBoardOrder :exec
UPDATE boards SET sort_order = $2 WHERE id = $1;

-- Template board columns (now in unified columns table)

-- name: ListTemplateBoardColumns :many
SELECT id, board_id, name, system_type, wip_limit, sort_order, is_locked, note
FROM columns
WHERE board_id = $1
ORDER BY sort_order ASC;

-- name: GetTemplateBoardColumnByID :one
SELECT id, board_id, name, system_type, wip_limit, sort_order, is_locked, note
FROM columns
WHERE id = $1;

-- name: CreateTemplateBoardColumn :one
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked, note)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, board_id, name, system_type, wip_limit, sort_order, is_locked, note;

-- name: UpdateTemplateBoardColumn :one
UPDATE columns
SET name = $2, system_type = $3, wip_limit = $4, note = $5
WHERE id = $1
RETURNING id, board_id, name, system_type, wip_limit, sort_order, is_locked, note;

-- name: DeleteTemplateBoardColumnByID :exec
DELETE FROM columns WHERE id = $1;

-- name: DeleteTemplateBoardColumnsByBoardID :exec
DELETE FROM columns WHERE board_id = $1;

-- name: UpdateTemplateBoardColumnOrder :exec
UPDATE columns SET sort_order = $2 WHERE id = $1;

-- Template board swimlanes (now in unified swimlanes table)

-- name: ListTemplateBoardSwimlanes :many
SELECT id, board_id, name, wip_limit, sort_order, note
FROM swimlanes
WHERE board_id = $1
ORDER BY sort_order ASC;

-- name: GetTemplateBoardSwimlaneByID :one
SELECT id, board_id, name, wip_limit, sort_order, note
FROM swimlanes
WHERE id = $1;

-- name: CreateTemplateBoardSwimlane :one
INSERT INTO swimlanes (board_id, name, wip_limit, sort_order, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, board_id, name, wip_limit, sort_order, note;

-- name: UpdateTemplateBoardSwimlane :one
UPDATE swimlanes
SET wip_limit = $2, note = $3
WHERE id = $1
RETURNING id, board_id, name, wip_limit, sort_order, note;

-- name: DeleteTemplateBoardSwimlaneByID :exec
DELETE FROM swimlanes WHERE id = $1;

-- name: DeleteTemplateBoardSwimlanesByBoardID :exec
DELETE FROM swimlanes WHERE board_id = $1;

-- name: UpdateTemplateBoardSwimlaneOrder :exec
UPDATE swimlanes SET sort_order = $2 WHERE id = $1;

-- Template board custom fields (now in unified fields table)

-- name: ListTemplateBoardFields :many
SELECT id, board_id, name, description, field_type, is_system, is_required, options
FROM fields
WHERE board_id = $1 AND kind = 'board_field';

-- name: ListTemplateBoardCustomFields :many
SELECT id, board_id, name, description, field_type, is_system, is_required, options
FROM fields
WHERE board_id = $1 AND kind = 'board_field' AND is_system = false;

-- name: GetTemplateBoardFieldByID :one
SELECT id, board_id, name, description, field_type, is_system, is_required, options
FROM fields
WHERE id = $1;

-- name: CreateTemplateBoardField :one
INSERT INTO fields (kind, board_id, name, description, field_type, is_system, is_required, options)
VALUES ('board_field', $1, $2, $3, $4, $5, $6, $7)
RETURNING id, board_id, name, description, field_type, is_system, is_required, options;

-- name: UpdateTemplateBoardField :one
UPDATE fields
SET name = $2, is_required = $3, options = $4
WHERE id = $1
RETURNING id, board_id, name, description, field_type, is_system, is_required, options;

-- name: DeleteTemplateBoardFieldByID :exec
DELETE FROM fields WHERE id = $1;

-- name: DeleteTemplateBoardFieldsByBoardID :exec
DELETE FROM fields WHERE board_id = $1;

-- name: DeleteNonSystemFieldsByBoardID :exec
DELETE FROM fields WHERE board_id = $1 AND is_system = false;
