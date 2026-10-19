ALTER TABLE journal_runners
    DROP COLUMN IF EXISTS strategy,
    DROP COLUMN IF EXISTS rules;

ALTER TABLE journals
    DROP COLUMN IF EXISTS strategy,
    DROP COLUMN IF EXISTS rules;
