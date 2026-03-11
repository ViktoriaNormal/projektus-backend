-- User-level blocks for login rate limiting
CREATE TABLE blocked_users (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    blocked_until TIMESTAMPTZ NOT NULL
);

