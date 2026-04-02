-- Tags for tasks within a board

-- name: ListTagsByBoard :many
SELECT id, board_id, name, created_at
FROM tags
WHERE board_id = $1
ORDER BY name;

-- name: GetTagByID :one
SELECT id, board_id, name, created_at
FROM tags
WHERE id = $1;

-- name: GetTagByBoardAndName :one
SELECT id, board_id, name, created_at
FROM tags
WHERE board_id = $1 AND name = $2;

-- name: CreateTag :one
INSERT INTO tags (board_id, name)
VALUES ($1, $2)
RETURNING id, board_id, name, created_at;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- Task-tag assignments

-- name: ListTaskTags :many
SELECT t.id, t.board_id, t.name, t.created_at
FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = $1
ORDER BY t.name;

-- name: AddTagToTask :exec
INSERT INTO task_tags (task_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromTask :exec
DELETE FROM task_tags
WHERE task_id = $1 AND tag_id = $2;

-- name: RemoveAllTagsFromTask :exec
DELETE FROM task_tags
WHERE task_id = $1;

-- name: ListTagsByTaskIDs :many
SELECT tt.task_id, t.id, t.board_id, t.name
FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = ANY($1::uuid[])
ORDER BY t.name;

-- name: CountTasksWithTag :one
SELECT COUNT(*)::int AS count
FROM task_tags
WHERE tag_id = $1;
