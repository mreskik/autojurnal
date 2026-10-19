-- Guard dobel: satu runner hanya boleh punya 1 trade open per pair + position di satu jurnal.
-- Input manual user (runner_id null) tidak dibatasi.
CREATE UNIQUE INDEX IF NOT EXISTS trades_runner_open_unique
    ON trades (journal_id, runner_id, pair, position)
    WHERE status = 'open' AND deleted_at IS NULL AND runner_id IS NOT NULL;
