-- Admin: list all users (with pagination, optional include deleted)
-- name: ListAllUsers :many
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at
FROM users
WHERE ($3::boolean IS TRUE OR deleted_at IS NULL)
ORDER BY username ASC
LIMIT $1 OFFSET $2;

-- name: ListAllUsersCount :one
SELECT COUNT(*) FROM users
WHERE ($1::boolean IS TRUE OR deleted_at IS NULL);

-- Admin: get user by ID (including deleted)
-- name: AdminGetUserByID :one
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at
FROM users
WHERE id = $1;

-- Admin: create user with position and status fields
-- name: AdminCreateUser :one
INSERT INTO users (username, email, password_hash, full_name, avatar_url, position, is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, username, email, password_hash, full_name, avatar_url, position,
          is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at;

-- Admin: update user fields
-- name: AdminUpdateUser :one
UPDATE users
SET username   = COALESCE(NULLIF(sqlc.arg(username)::text, ''), username),
    email      = COALESCE(NULLIF(sqlc.arg(email)::text, ''), email),
    full_name  = COALESCE(NULLIF(sqlc.arg(full_name)::text, ''), full_name),
    position   = CASE WHEN sqlc.arg(set_position)::boolean THEN sqlc.arg(position) ELSE position END,
    is_active  = CASE WHEN sqlc.arg(set_is_active)::boolean THEN sqlc.arg(is_active)::boolean ELSE is_active END,
    on_vacation = CASE WHEN sqlc.arg(set_on_vacation)::boolean THEN sqlc.arg(on_vacation)::boolean ELSE on_vacation END,
    is_sick     = CASE WHEN sqlc.arg(set_is_sick)::boolean THEN sqlc.arg(is_sick)::boolean ELSE is_sick END,
    alt_contact_channel = CASE WHEN sqlc.arg(set_alt_contact_channel)::boolean THEN sqlc.arg(alt_contact_channel) ELSE alt_contact_channel END,
    alt_contact_info    = CASE WHEN sqlc.arg(set_alt_contact_info)::boolean THEN sqlc.arg(alt_contact_info) ELSE alt_contact_info END
WHERE id = sqlc.arg(id)
RETURNING id, username, email, password_hash, full_name, avatar_url, position,
          is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at;

-- Admin: soft delete user (deactivate and set deleted_at)
-- name: SoftDeleteUser :exec
UPDATE users
SET is_active = false,
    deleted_at = NOW()
WHERE id = $1;
