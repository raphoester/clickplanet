-- 0: recorded before the rank was kept.
ALTER TABLE finale_fights ADD COLUMN tiles_rank integer NOT NULL DEFAULT 0 CHECK (tiles_rank >= 0);
