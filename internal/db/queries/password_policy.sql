-- Текущая парольная политика (последняя по дате обновления)
-- name: GetCurrentPasswordPolicy :one
SELECT id, min_length, require_digits, require_lowercase, require_uppercase, require_special, notes, updated_at, updated_by
FROM password_policy
ORDER BY updated_at DESC
LIMIT 1;

-- Добавить новую версию политики (актуальной считается последняя)
-- name: InsertPasswordPolicy :one
INSERT INTO password_policy (min_length, require_digits, require_lowercase, require_uppercase, require_special, notes, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, min_length, require_digits, require_lowercase, require_uppercase, require_special, notes, updated_at, updated_by;
