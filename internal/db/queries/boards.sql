-- Boards

-- name: CreateBoard :one
INSERT INTO boards (project_id, template_id, name, description, sort_order, is_default, priority_type, estimation_unit, swimlane_group_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, project_id, template_id, name, description, sort_order, priority_type, estimation_unit, swimlane_group_by, is_default;

-- name: GetBoardByID :one
SELECT id, project_id, template_id, name, description, sort_order, priority_type, estimation_unit, swimlane_group_by, is_default
FROM boards
WHERE id = $1;

-- name: ListProjectBoards :many
SELECT id, project_id, template_id, name, description, sort_order, priority_type, estimation_unit, swimlane_group_by, is_default
FROM boards
WHERE project_id = $1
ORDER BY sort_order;

-- name: UpdateBoard :one
UPDATE boards
SET name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    sort_order  = COALESCE(sqlc.narg('sort_order'), sort_order),
    priority_type    = COALESCE(sqlc.narg('priority_type'), priority_type),
    estimation_unit  = COALESCE(sqlc.narg('estimation_unit'), estimation_unit),
    swimlane_group_by = COALESCE(sqlc.narg('swimlane_group_by'), swimlane_group_by)
WHERE id = sqlc.arg('id')
RETURNING id, project_id, template_id, name, description, sort_order, priority_type, estimation_unit, swimlane_group_by, is_default;

-- name: DeleteBoard :exec
DELETE FROM boards
WHERE id = $1;

-- name: UpdateBoardOrder :exec
UPDATE boards SET sort_order = $2 WHERE id = $1;

-- Columns

-- name: ListBoardColumns :many
SELECT id, board_id, name, system_type, wip_limit, sort_order, is_locked, note
FROM columns
WHERE board_id = $1
ORDER BY sort_order;

-- name: GetColumnByID :one
SELECT id, board_id, name, system_type, wip_limit, sort_order, is_locked, note
FROM columns
WHERE id = $1;

-- name: CreateColumn :one
INSERT INTO columns (board_id, name, system_type, wip_limit, sort_order, is_locked)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, board_id, name, system_type, wip_limit, sort_order, is_locked, note;

-- name: UpdateColumn :one
UPDATE columns
SET name        = COALESCE(sqlc.narg('name'), name),
    system_type = COALESCE(sqlc.narg('system_type'), system_type),
    wip_limit   = COALESCE(sqlc.narg('wip_limit'), wip_limit),
    sort_order  = COALESCE(sqlc.narg('sort_order'), sort_order)
WHERE id = sqlc.arg('id')
RETURNING id, board_id, name, system_type, wip_limit, sort_order, is_locked, note;

-- name: DeleteColumn :exec
DELETE FROM columns
WHERE id = $1;

-- name: UpdateColumnOrder :exec
UPDATE columns SET sort_order = $2 WHERE id = $1;

-- name: CountTasksInColumn :one
SELECT COUNT(*)::int AS count FROM tasks WHERE column_id = $1;

-- Swimlanes

-- name: ListBoardSwimlanes :many
SELECT id, board_id, name, wip_limit, sort_order, note
FROM swimlanes
WHERE board_id = $1
ORDER BY sort_order;

-- name: GetSwimlaneByID :one
SELECT id, board_id, name, wip_limit, sort_order, note
FROM swimlanes
WHERE id = $1;

-- name: CreateSwimlane :one
INSERT INTO swimlanes (board_id, name, wip_limit, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, board_id, name, wip_limit, sort_order, note;

-- name: UpdateSwimlane :one
UPDATE swimlanes
SET name       = COALESCE(sqlc.narg('name'), name),
    wip_limit  = COALESCE(sqlc.narg('wip_limit'), wip_limit),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order)
WHERE id = sqlc.arg('id')
RETURNING id, board_id, name, wip_limit, sort_order, note;

-- name: DeleteSwimlane :exec
DELETE FROM swimlanes
WHERE id = $1;

-- name: UpdateSwimlaneOrder :exec
UPDATE swimlanes SET sort_order = $2 WHERE id = $1;

-- name: CountTasksInSwimlane :one
SELECT COUNT(*)::int AS count FROM tasks WHERE swimlane_id = $1;

-- Notes

-- name: ListBoardNotes :many
SELECT n.id, n.column_id, n.swimlane_id, n.content
FROM notes n
JOIN columns c ON n.column_id = c.id
WHERE c.board_id = $1
UNION ALL
SELECT n.id, n.column_id, n.swimlane_id, n.content
FROM notes n
JOIN swimlanes s ON n.swimlane_id = s.id
WHERE s.board_id = $1;

-- name: GetNoteByID :one
SELECT id, column_id, swimlane_id, content
FROM notes
WHERE id = $1;

-- name: CreateNoteForColumn :one
INSERT INTO notes (column_id, content)
VALUES ($1, $2)
RETURNING id, column_id, swimlane_id, content;

-- name: CreateNoteForSwimlane :one
INSERT INTO notes (swimlane_id, content)
VALUES ($1, $2)
RETURNING id, column_id, swimlane_id, content;

-- name: UpdateNote :one
UPDATE notes
SET content = COALESCE(sqlc.narg('content'), content)
WHERE id = sqlc.arg('id')
RETURNING id, column_id, swimlane_id, content;

-- name: DeleteNote :exec
DELETE FROM notes
WHERE id = $1;
