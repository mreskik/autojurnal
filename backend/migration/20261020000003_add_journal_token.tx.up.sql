-- Token jurnal: dipakai runner (header X-Journal-Token) bersama token runner.
ALTER TABLE journals ADD COLUMN IF NOT EXISTS token VARCHAR(64);

-- isi token untuk jurnal yang sudah ada (40 karakter hex acak)
UPDATE journals
SET token = substr(replace(gen_random_uuid()::text || gen_random_uuid()::text, '-', ''), 1, 40)
WHERE token IS NULL;

ALTER TABLE journals ALTER COLUMN token SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS journals_token_unique ON journals (token);
