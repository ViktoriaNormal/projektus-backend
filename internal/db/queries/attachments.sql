-- Attachments for tasks and comments

-- name: CreateTaskAttachment :one
INSERT INTO attachments (task_id, file_name, file_path, uploaded_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateCommentAttachment :one
INSERT INTO attachments (comment_id, file_name, file_path, uploaded_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTaskAttachments :many
SELECT *
FROM attachments
WHERE task_id = $1
ORDER BY uploaded_at DESC;

-- name: ListCommentAttachments :many
SELECT *
FROM attachments
WHERE comment_id = $1
ORDER BY uploaded_at DESC;

