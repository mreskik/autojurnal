DROP INDEX IF EXISTS journals_token_unique;
ALTER TABLE journals DROP COLUMN IF EXISTS token;
