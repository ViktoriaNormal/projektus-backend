-- Task comments and mentions

-- name: CreateComment :one
INSERT INTO comments (task_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTaskComments :many
SELECT *
FROM comments
WHERE task_id = $1
ORDER BY created_at;

-- name: CreateCommentMention :exec
INSERT INTO comment_mentions (comment_id, project_member_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListCommentMentions :many
SELECT *
FROM comment_mentions
WHERE comment_id = $1;

