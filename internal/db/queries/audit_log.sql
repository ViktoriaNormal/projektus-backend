-- Список записей журнала с фильтрами (user_id, action_type, from, to опциональны)
-- name: ListAuditLogs :many
SELECT *
FROM audit_log
WHERE (sqlc.narg('user_id') IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('action_type') IS NULL OR sqlc.narg('action_type') = '' OR action_type = sqlc.narg('action_type'))
  AND (sqlc.narg('from_at') IS NULL OR created_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at') IS NULL OR created_at <= sqlc.narg('to_at'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- Количество записей по тем же фильтрам (для пагинации)
-- name: CountAuditLogs :one
SELECT COUNT(*)
FROM audit_log
WHERE (sqlc.narg('user_id') IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('action_type') IS NULL OR sqlc.narg('action_type') = '' OR action_type = sqlc.narg('action_type'))
  AND (sqlc.narg('from_at') IS NULL OR created_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at') IS NULL OR created_at <= sqlc.narg('to_at'));

-- Добавить запись в журнал
-- name: InsertAuditLog :one
INSERT INTO audit_log (user_id, action_type, entity_type, entity_id, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
