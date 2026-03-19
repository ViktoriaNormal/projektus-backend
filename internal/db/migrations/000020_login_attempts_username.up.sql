-- Rename email column to username in login_attempts (auth by username, not email)
ALTER TABLE login_attempts RENAME COLUMN email TO username;

-- Recreate index with new column name
DROP INDEX IF EXISTS idx_login_attempts_email_attempted_at;
CREATE INDEX idx_login_attempts_username_attempted_at ON login_attempts (username, attempted_at);
