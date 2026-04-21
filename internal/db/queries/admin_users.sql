-- Admin: list all users с пагинацией и серверными фильтрами.
-- name: ListAllUsers :many
-- include_deleted=true подключает soft-deleted; q (ILIKE по username/email/full_name/position),
-- is_active_filter (true/false) и role_id_filter — опциональны, пропуск фильтра = nil.
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users u
WHERE (sqlc.arg('include_deleted')::boolean IS TRUE OR u.deleted_at IS NULL)
  AND (sqlc.narg('q')::text IS NULL OR sqlc.narg('q')::text = '' OR (
       u.username ILIKE '%' || sqlc.narg('q') || '%'
    OR u.email    ILIKE '%' || sqlc.narg('q') || '%'
    OR u.full_name ILIKE '%' || sqlc.narg('q') || '%'
    OR COALESCE(u.position, '') ILIKE '%' || sqlc.narg('q') || '%'))
  AND (sqlc.narg('is_active_filter')::boolean IS NULL OR u.is_active = sqlc.narg('is_active_filter')::boolean)
  AND (sqlc.narg('role_id_filter')::uuid IS NULL OR EXISTS (
       SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id_filter')::uuid))
ORDER BY LOWER(u.full_name) ASC, u.id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: ListAllUsersCount :one
-- Полное число записей под применённые фильтры (без учёта limit/offset).
SELECT COUNT(*)::bigint FROM users u
WHERE (sqlc.arg('include_deleted')::boolean IS TRUE OR u.deleted_at IS NULL)
  AND (sqlc.narg('q')::text IS NULL OR sqlc.narg('q')::text = '' OR (
       u.username ILIKE '%' || sqlc.narg('q') || '%'
    OR u.email    ILIKE '%' || sqlc.narg('q') || '%'
    OR u.full_name ILIKE '%' || sqlc.narg('q') || '%'
    OR COALESCE(u.position, '') ILIKE '%' || sqlc.narg('q') || '%'))
  AND (sqlc.narg('is_active_filter')::boolean IS NULL OR u.is_active = sqlc.narg('is_active_filter')::boolean)
  AND (sqlc.narg('role_id_filter')::uuid IS NULL OR EXISTS (
       SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id_filter')::uuid));

-- name: CountActiveUsers :one
-- Число активных пользователей по всему множеству (is_active=true).
-- include_deleted управляет учётом soft-deleted. Независимо от q/is_active_filter/role_id —
-- чтобы карточка «Активные» не менялась при фильтрации в таблице.
SELECT COUNT(*)::bigint FROM users
WHERE is_active = true
  AND ($1::boolean IS TRUE OR deleted_at IS NULL);

-- name: CountInactiveUsers :one
-- Число заблокированных/деактивированных пользователей по всему множеству (is_active=false).
SELECT COUNT(*)::bigint FROM users
WHERE is_active = false
  AND ($1::boolean IS TRUE OR deleted_at IS NULL);

-- Admin: get user by ID (including deleted)
-- name: AdminGetUserByID :one
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users
WHERE id = $1;

-- Admin: create user with position and status fields
-- name: AdminCreateUser :one
INSERT INTO users (username, email, password_hash, full_name, avatar_url, position, is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, username, email, password_hash, full_name, avatar_url, position,
          is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until;

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
          is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until;

-- Admin: soft delete user (deactivate and set deleted_at)
-- name: SoftDeleteUser :exec
UPDATE users
SET is_active = false,
    deleted_at = NOW()
WHERE id = $1;
