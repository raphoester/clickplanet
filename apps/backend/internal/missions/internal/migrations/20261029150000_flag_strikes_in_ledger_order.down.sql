ALTER TABLE flag_days
    ADD COLUMN counted_strikes_until timestamptz NOT NULL DEFAULT 'epoch',
    DROP COLUMN counted_strikes_below,
    DROP COLUMN counted_strikes_below_index;

ALTER TABLE flag_days ALTER COLUMN counted_strikes_until DROP DEFAULT;
