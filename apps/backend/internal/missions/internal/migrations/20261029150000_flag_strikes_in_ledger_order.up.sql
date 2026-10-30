ALTER TABLE flag_days
    ADD COLUMN counted_strikes_below       bigint  NOT NULL DEFAULT 0 CHECK (counted_strikes_below >= 0),
    ADD COLUMN counted_strikes_below_index integer NOT NULL DEFAULT 0 CHECK (counted_strikes_below_index >= 0),
    DROP COLUMN counted_strikes_until;

ALTER TABLE flag_days
    ALTER COLUMN counted_strikes_below DROP DEFAULT,
    ALTER COLUMN counted_strikes_below_index DROP DEFAULT;
