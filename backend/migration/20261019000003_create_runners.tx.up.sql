CREATE TABLE IF NOT EXISTS runners (
    id           UUID         PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    token        VARCHAR(64)  NOT NULL,
    is_active    BOOLEAN      NOT NULL DEFAULT true,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS runners_token_unique ON runners (token);
CREATE INDEX IF NOT EXISTS runners_user_id_idx ON runners (user_id) WHERE deleted_at IS NULL;
