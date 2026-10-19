CREATE TABLE IF NOT EXISTS journal_transactions (
    id          UUID          PRIMARY KEY DEFAULT uuidv7(),
    journal_id  UUID          NOT NULL REFERENCES journals (id) ON DELETE CASCADE,
    type        VARCHAR(10)   NOT NULL CHECK (type IN ('deposit', 'withdraw')),
    amount      NUMERIC(20,8) NOT NULL CHECK (amount > 0),
    note        TEXT,
    occurred_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS journal_transactions_journal_idx
    ON journal_transactions (journal_id, occurred_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS journal_runners (
    journal_id UUID        NOT NULL REFERENCES journals (id) ON DELETE CASCADE,
    runner_id  UUID        NOT NULL REFERENCES runners (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (journal_id, runner_id)
);

CREATE INDEX IF NOT EXISTS journal_runners_runner_idx ON journal_runners (runner_id);
