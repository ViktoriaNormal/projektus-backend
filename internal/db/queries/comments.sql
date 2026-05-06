-- name: ListTaskComments :many
SELECT id, task_id, author_id, content, parent_comment_id, created_at, updated_at
FROM comments
WHERE task_id = $1
ORDER BY created_at ASC;

-- name: CreateComment :one
INSERT INTO comments (task_id, author_id, content, parent_comment_id)
VALUES ($1, $2, $3, $4)
RETURNING id, task_id, author_id, content, parent_comment_id, created_at, updated_at;

-- name: UpdateComment :one
UPDATE comments
SET content = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, task_id, author_id, content, parent_comment_id, created_at, updated_at;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: GetCommentByID :one
SELECT id, task_id, author_id, content, parent_comment_id, created_at, updated_at
FROM comments
WHERE id = $1;
