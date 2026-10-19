CREATE TABLE IF NOT EXISTS journals (
    id              UUID          PRIMARY KEY DEFAULT uuidv7(),
    user_id         UUID          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            VARCHAR(100)  NOT NULL,
    description     TEXT,
    initial_balance NUMERIC(20,8) NOT NULL CHECK (initial_balance > 0),
    period_type     VARCHAR(10)   NOT NULL CHECK (period_type IN ('daily', 'weekly', 'monthly')),
    timezone        VARCHAR(64)   NOT NULL DEFAULT 'Asia/Jakarta',
    start_date      DATE          NOT NULL,
    status          VARCHAR(10)   NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS journals_user_id_idx ON journals (user_id) WHERE deleted_at IS NULL;

-- Riwayat target: hanya insert, tidak pernah di-update.
-- Target yang berlaku untuk suatu periode = effective_from terbesar yang <= awal periode.
CREATE TABLE IF NOT EXISTS journal_targets (
    id             UUID          PRIMARY KEY DEFAULT uuidv7(),
    journal_id     UUID          NOT NULL REFERENCES journals (id) ON DELETE CASCADE,
    target_type    VARCHAR(10)   NOT NULL CHECK (target_type IN ('amount', 'percent')),
    target_value   NUMERIC(20,8) NOT NULL CHECK (target_value > 0),
    effective_from DATE          NOT NULL,
    note           TEXT,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS journal_targets_journal_idx ON journal_targets (journal_id, effective_from, id);
