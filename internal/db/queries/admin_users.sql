-- Admin: list all users (with pagination, optional include deleted)
-- name: ListAllUsers :many
SELECT *
FROM users
WHERE ($3::boolean IS TRUE OR deleted_at IS NULL)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAllUsersCount :one
SELECT COUNT(*) FROM users
WHERE ($1::boolean IS TRUE OR deleted_at IS NULL);

-- Admin: soft delete user (deactivate and set deleted_at)
-- name: SoftDeleteUser :exec
UPDATE users
SET is_active = false,
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1;
