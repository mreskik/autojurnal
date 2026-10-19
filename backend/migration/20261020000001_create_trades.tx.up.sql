CREATE TABLE IF NOT EXISTS trades (
    id           UUID          PRIMARY KEY DEFAULT uuidv7(),
    journal_id   UUID          NOT NULL REFERENCES journals (id) ON DELETE CASCADE,
    runner_id    UUID          REFERENCES runners (id) ON DELETE SET NULL, -- null = input manual user
    exchange     VARCHAR(30),
    pair         VARCHAR(30)   NOT NULL,
    position     VARCHAR(5)    NOT NULL CHECK (position IN ('long', 'short')),
    margin       NUMERIC(20,8) NOT NULL CHECK (margin > 0),
    leverage     NUMERIC(10,2) NOT NULL CHECK (leverage >= 1),
    entry_price  NUMERIC(30,10) NOT NULL CHECK (entry_price > 0),
    exit_price   NUMERIC(30,10) CHECK (exit_price > 0),
    stop_loss    NUMERIC(30,10) CHECK (stop_loss > 0),
    take_profit  NUMERIC(30,10) CHECK (take_profit > 0),
    status       VARCHAR(6)    NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    open_reason  TEXT,
    close_reason VARCHAR(20)   CHECK (close_reason IN ('take_profit', 'stop_loss', 'manual', 'liquidation', 'trailing_stop')),
    close_note   TEXT,
    pnl          NUMERIC(20,8),               -- PnL bersih (yang masuk balance), diisi saat close
    fee          NUMERIC(20,8) NOT NULL DEFAULT 0 CHECK (fee >= 0),
    opened_at    TIMESTAMPTZ   NOT NULL,
    closed_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,

    -- open: belum ada data close sama sekali; closed: data close wajib lengkap
    CONSTRAINT trades_status_consistency CHECK (
        (status = 'open' AND exit_price IS NULL AND closed_at IS NULL AND pnl IS NULL AND close_reason IS NULL)
        OR
        (status = 'closed' AND exit_price IS NOT NULL AND closed_at IS NOT NULL AND pnl IS NOT NULL AND close_reason IS NOT NULL)
    ),
    CONSTRAINT trades_closed_after_opened CHECK (closed_at IS NULL OR closed_at >= opened_at)
);

CREATE INDEX IF NOT EXISTS trades_journal_opened_idx ON trades (journal_id, opened_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS trades_journal_closed_idx ON trades (journal_id, closed_at) WHERE deleted_at IS NULL AND status = 'closed';
CREATE INDEX IF NOT EXISTS trades_runner_idx ON trades (runner_id) WHERE deleted_at IS NULL;
