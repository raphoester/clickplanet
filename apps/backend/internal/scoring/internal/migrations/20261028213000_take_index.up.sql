ALTER TABLE contributions
    ADD COLUMN counted_takes_below_index integer NOT NULL DEFAULT 0 CHECK (counted_takes_below_index >= 0);
ALTER TABLE rounds
    ADD COLUMN counted_takes_below_index integer NOT NULL DEFAULT 0 CHECK (counted_takes_below_index >= 0);
