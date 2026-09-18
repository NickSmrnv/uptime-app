CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    name varchar(100) NOT NULL,
    email varchar(320) NOT NULL,
    avatar_filename varchar(80),
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS name varchar(100) NOT NULL DEFAULT 'Пользователь';
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_filename varchar(80);
ALTER TABLE users
    ALTER COLUMN name DROP DEFAULT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

UPDATE users
SET avatar_filename = 'avatars/' || avatar_filename
WHERE avatar_filename <> ''
  AND position('/' IN avatar_filename) = 0;

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
    token_hash varchar(64) NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    replaced_by_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id ON refresh_sessions (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_sessions_token_hash ON refresh_sessions (token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_sessions_expires_at ON refresh_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_sessions_revoked_at ON refresh_sessions (revoked_at);

CREATE TABLE IF NOT EXISTS monitors (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
    url varchar(2048) NOT NULL,
    interval_seconds integer NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitors_user_created ON monitors (user_id, created_at);
