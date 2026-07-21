CREATE TABLE IF NOT EXISTS user_logins (
    id UUID PRIMARY KEY gen_random_uuid(),
    user_id UUID NOT NULL,
    login_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_logins_unique ON user_logins (user_id, login_time);

CREATE INDEX IF NOT EXISTS idx_user_logins_login_time ON user_logins (login_time);