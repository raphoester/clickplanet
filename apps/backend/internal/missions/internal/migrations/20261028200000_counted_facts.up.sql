ALTER TABLE mission_sheets
    ADD COLUMN counted_bombs_below     bigint NOT NULL DEFAULT 0 CHECK (counted_bombs_below >= 0),
    ADD COLUMN counted_shields_below   bigint NOT NULL DEFAULT 0 CHECK (counted_shields_below >= 0),
    ADD COLUMN counted_fortifies_below bigint NOT NULL DEFAULT 0 CHECK (counted_fortifies_below >= 0),
    ADD COLUMN counted_boxes_until     timestamptz,
    ADD COLUMN counted_answers_until   timestamptz;
