-- The seasons, one after the other: each starts its break after the end of the one before, season 0's from its origin
-- (season.Origin, 2026-10-03, the day its countdown shipped). Season 0 was scoring's YAML end, 2026-10-31 23:00 UTC.
CREATE TABLE seasons (
    number       bigint   PRIMARY KEY CHECK (number >= 0 AND number <= 4294967295),
    break_before interval NOT NULL CHECK (break_before >= interval '0'),
    length       interval NOT NULL CHECK (length > interval '0')
);

INSERT INTO seasons (number, break_before, length) VALUES (0, interval '0', interval '28 days 23 hours');
