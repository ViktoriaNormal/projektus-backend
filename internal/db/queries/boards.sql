-- Boards

-- name: CreateBoard :one
INSERT INTO boards (project_id, template_id, name, description, "order")
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBoardByID :one
SELECT *
FROM boards
WHERE id = $1;

-- name: ListProjectBoards :many
SELECT *
FROM boards
WHERE project_id = $1
ORDER BY "order";

-- name: UpdateBoard :one
UPDATE boards
SET name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    "order"     = COALESCE(sqlc.narg('order'), "order"),
    updated_at  = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteBoard :exec
DELETE FROM boards
WHERE id = $1;

-- Columns

-- name: ListBoardColumns :many
SELECT *
FROM columns
WHERE board_id = $1
ORDER BY "order";

-- name: CreateColumn :one
INSERT INTO columns (board_id, name, system_type, wip_limit, "order")
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateColumn :one
UPDATE columns
SET name        = COALESCE(sqlc.narg('name'), name),
    system_type = COALESCE(sqlc.narg('system_type'), system_type),
    wip_limit   = COALESCE(sqlc.narg('wip_limit'), wip_limit),
    "order"     = COALESCE(sqlc.narg('order'), "order"),
    updated_at  = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteColumn :exec
DELETE FROM columns
WHERE id = $1;

-- Swimlanes

-- name: ListBoardSwimlanes :many
SELECT *
FROM swimlanes
WHERE board_id = $1
ORDER BY "order";

-- name: CreateSwimlane :one
INSERT INTO swimlanes (board_id, name, wip_limit, "order")
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSwimlane :one
UPDATE swimlanes
SET name       = COALESCE(sqlc.narg('name'), name),
    wip_limit  = COALESCE(sqlc.narg('wip_limit'), wip_limit),
    "order"    = COALESCE(sqlc.narg('order'), "order"),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSwimlane :exec
DELETE FROM swimlanes
WHERE id = $1;

-- Notes

-- name: ListBoardNotes :many
SELECT n.*
FROM notes n
JOIN columns c ON n.column_id = c.id
WHERE c.board_id = $1
UNION ALL
SELECT n.*
FROM notes n
JOIN swimlanes s ON n.swimlane_id = s.id
WHERE s.board_id = $1;

-- name: CreateNoteForColumn :one
INSERT INTO notes (column_id, content)
VALUES ($1, $2)
RETURNING *;

-- name: CreateNoteForSwimlane :one
INSERT INTO notes (swimlane_id, content)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateNote :one
UPDATE notes
SET content    = COALESCE(sqlc.narg('content'), content),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteNote :exec
DELETE FROM notes
WHERE id = $1;

