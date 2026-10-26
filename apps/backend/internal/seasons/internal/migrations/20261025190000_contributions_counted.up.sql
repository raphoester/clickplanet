ALTER TABLE contributions ADD COLUMN counted_takes_below bigint NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0);
