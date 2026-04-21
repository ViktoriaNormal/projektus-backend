-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, username, email, password_hash, full_name, avatar_url, position,
          is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users
WHERE username = $1;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: InsertPasswordHistory :exec
INSERT INTO password_history (user_id, password_hash)
VALUES ($1, $2);

-- name: GetLastNPasswordHashes :many
SELECT password_hash
FROM password_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: UpdateUserProfile :exec
UPDATE users
SET full_name = $2,
    email     = $3,
    position  = $4,
    on_vacation = $5,
    is_sick     = $6,
    alt_contact_channel = $7,
    alt_contact_info    = $8
WHERE id = $1;

-- name: UpdateUserAvatar :exec
UPDATE users
SET avatar_url = $2
WHERE id = $1;

-- name: SearchUsers :many
-- Порядок: ORDER BY LOWER(full_name) ASC (регистро-независимо, кириллица
-- сортируется корректно), id ASC как вторичный ключ — стабильный
-- tiebreaker для равных ФИО, чтобы пагинация не «скакала».
SELECT id, username, email, password_hash, full_name, avatar_url, position,
       is_active, on_vacation, is_sick, alt_contact_channel, alt_contact_info, deleted_at, blocked_until
FROM users
WHERE deleted_at IS NULL
  AND ($1::text IS NULL OR $1::text = '' OR (
   username ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%'
   OR full_name ILIKE '%' || $1 || '%'
   OR position ILIKE '%' || $1 || '%'
   OR alt_contact_info ILIKE '%' || $1 || '%'))
ORDER BY LOWER(full_name) ASC, id ASC
LIMIT $2 OFFSET $3;

-- name: CountSearchUsers :one
-- Полное количество пользователей, подходящих под фильтр q — без учёта limit/offset.
-- Используется эндпоинтом GET /users для возврата поля `total` рядом с массивом
-- пользователей.
SELECT COUNT(*)::bigint
FROM users
WHERE deleted_at IS NULL
  AND ($1::text IS NULL OR $1::text = '' OR (
   username ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%'
   OR full_name ILIKE '%' || $1 || '%'
   OR position ILIKE '%' || $1 || '%'
   OR alt_contact_info ILIKE '%' || $1 || '%'));

-- name: ListAllUserIDs :many
SELECT id
FROM users;
