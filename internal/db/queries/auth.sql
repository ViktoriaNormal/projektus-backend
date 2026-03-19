-- Refresh tokens

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1
  AND revoked_at IS NULL;

-- Login attempts

-- name: InsertLoginAttempt :exec
INSERT INTO login_attempts (username, ip_address, success)
VALUES ($1, $2, $3);

-- name: CountFailedAttemptsByUsernameSince :one
SELECT COUNT(*)::INT
FROM login_attempts
WHERE username = $1
  AND success = FALSE
  AND attempted_at >= $2;

-- name: CountFailedAttemptsByIPSince :one
SELECT COUNT(*)::INT
FROM login_attempts
WHERE ip_address = $1
  AND success = FALSE
  AND attempted_at >= $2;

-- Blocked IPs

-- name: GetBlockedIP :one
SELECT ip_address, blocked_until
FROM blocked_ips
WHERE ip_address = $1;

-- name: UpsertBlockedIP :exec
INSERT INTO blocked_ips (ip_address, blocked_until)
VALUES ($1, $2)
ON CONFLICT (ip_address) DO UPDATE
SET blocked_until = EXCLUDED.blocked_until;

-- name: DeleteExpiredBlockedIPs :exec
DELETE FROM blocked_ips
WHERE blocked_until <= NOW();

-- Blocked users

-- name: GetBlockedUser :one
SELECT user_id, blocked_until
FROM blocked_users
WHERE user_id = $1;

-- name: UpsertBlockedUser :exec
INSERT INTO blocked_users (user_id, blocked_until)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET blocked_until = EXCLUDED.blocked_until;

-- name: DeleteExpiredBlockedUsers :exec
DELETE FROM blocked_users
WHERE blocked_until <= NOW();

