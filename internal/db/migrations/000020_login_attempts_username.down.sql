ALTER TABLE login_attempts RENAME COLUMN username TO email;

DROP INDEX IF EXISTS idx_login_attempts_username_attempted_at;
CREATE INDEX idx_login_attempts_email_attempted_at ON login_attempts (email, attempted_at);
