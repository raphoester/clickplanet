-- Every day id so far is the gamemaster's round id, season-N/day-K: the day before a day is K - 1 of the same season,
-- even when a battle cut it short.
ALTER TABLE mission_days ADD COLUMN season integer, ADD COLUMN number integer;

UPDATE mission_days SET
    season = substring(day_id FROM '^season-([0-9]+)/day-[0-9]+$')::integer,
    number = substring(day_id FROM '^season-[0-9]+/day-([0-9]+)$')::integer;

ALTER TABLE mission_days
    ALTER COLUMN season SET NOT NULL,
    ALTER COLUMN number SET NOT NULL,
    ADD CHECK (season >= 0),
    ADD CHECK (number > 0),
    ADD UNIQUE (season, number);
