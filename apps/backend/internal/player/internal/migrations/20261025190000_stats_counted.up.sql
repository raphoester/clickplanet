ALTER TABLE stats
    ADD COLUMN counted_takes_below     bigint NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0),
    ADD COLUMN counted_bombs_below     bigint NOT NULL DEFAULT 0 CHECK (counted_bombs_below >= 0),
    ADD COLUMN counted_fortifies_below bigint NOT NULL DEFAULT 0 CHECK (counted_fortifies_below >= 0),
    ADD COLUMN counted_messages_until  timestamptz,
    ADD COLUMN counted_bonuses_until   timestamptz,
    ADD COLUMN counted_answers_until   timestamptz;
