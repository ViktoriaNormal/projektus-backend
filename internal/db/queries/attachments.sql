-- name: ListTaskAttachments :many
SELECT id, task_id, comment_id, file_name, file_path, file_size, content_type, uploaded_by, uploaded_at
FROM attachments
WHERE task_id = $1
ORDER BY uploaded_at ASC;

-- name: CreateAttachment :one
INSERT INTO attachments (task_id, comment_id, file_name, file_path, file_size, content_type, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, task_id, comment_id, file_name, file_path, file_size, content_type, uploaded_by, uploaded_at;

-- name: DeleteAttachment :exec
DELETE FROM attachments WHERE id = $1;

-- name: GetAttachmentByID :one
SELECT id, task_id, comment_id, file_name, file_path, file_size, content_type, uploaded_by, uploaded_at
FROM attachments
WHERE id = $1;
