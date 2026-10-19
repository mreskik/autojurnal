-- Instruksi teks untuk runner (panduan, tidak dipaksakan server).
-- journals         : berlaku untuk semua runner di jurnal
-- journal_runners  : khusus satu runner di jurnal
ALTER TABLE journals
    ADD COLUMN IF NOT EXISTS rules    TEXT,
    ADD COLUMN IF NOT EXISTS strategy TEXT;

ALTER TABLE journal_runners
    ADD COLUMN IF NOT EXISTS rules    TEXT,
    ADD COLUMN IF NOT EXISTS strategy TEXT;
