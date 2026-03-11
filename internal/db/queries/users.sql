-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at    = NOW()
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

