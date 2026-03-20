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

-- Admin: get user by ID (including deleted)
-- name: AdminGetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- Admin: create user with position
-- name: AdminCreateUser :one
INSERT INTO users (username, email, password_hash, full_name, avatar_url, position, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- Admin: update user fields
-- name: AdminUpdateUser :one
UPDATE users
SET username   = COALESCE(NULLIF(sqlc.arg(username)::text, ''), username),
    email      = COALESCE(NULLIF(sqlc.arg(email)::text, ''), email),
    full_name  = COALESCE(NULLIF(sqlc.arg(full_name)::text, ''), full_name),
    position   = CASE WHEN sqlc.arg(set_position)::boolean THEN sqlc.arg(position) ELSE position END,
    is_active  = CASE WHEN sqlc.arg(set_is_active)::boolean THEN sqlc.arg(is_active)::boolean ELSE is_active END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- Admin: soft delete user (deactivate and set deleted_at)
-- name: SoftDeleteUser :exec
UPDATE users
SET is_active = false,
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1;
