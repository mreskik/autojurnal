ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_goal_consistency;

ALTER TABLE journals
    DROP COLUMN IF EXISTS goal_achieved_at,
    DROP COLUMN IF EXISTS goal_value,
    DROP COLUMN IF EXISTS goal_type;
