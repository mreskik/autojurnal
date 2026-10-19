-- Target akhir jurnal (opsional), beda dengan target per periode di journal_targets.
ALTER TABLE journals
    ADD COLUMN IF NOT EXISTS goal_type        VARCHAR(10)   CHECK (goal_type IN ('balance', 'profit', 'percent')),
    ADD COLUMN IF NOT EXISTS goal_value       NUMERIC(20,8) CHECK (goal_value > 0),
    ADD COLUMN IF NOT EXISTS goal_achieved_at TIMESTAMPTZ;

ALTER TABLE journals
    ADD CONSTRAINT journals_goal_consistency CHECK (
        (goal_type IS NULL AND goal_value IS NULL AND goal_achieved_at IS NULL)
        OR (goal_type IS NOT NULL AND goal_value IS NOT NULL)
    );
