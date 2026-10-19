CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT uuidv7(),
    username      VARCHAR(50)  NOT NULL,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(255),
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique
    ON users (lower(username)) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique
    ON users (lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
