-- Refresh tokens

-- name: CreateRefreshToken :one
INSERT INTO tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, revoked_at;

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, revoked_at
FROM tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE tokens
SET revoked_at = NOW()
WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE tokens
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

-- Blocked users (temporary block via users.blocked_until)

-- name: GetUserBlockedUntil :one
SELECT blocked_until
FROM users
WHERE id = $1;

-- name: SetUserBlockedUntil :exec
UPDATE users
SET blocked_until = $2, is_active = false
WHERE id = $1;

-- name: ClearExpiredUserBlocks :exec
UPDATE users
SET blocked_until = NULL, is_active = true
WHERE blocked_until IS NOT NULL AND blocked_until <= NOW();

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
