-- The seasons, which the gamemaster tells scoring. Season 0 was scoring's calendar in the YAML, with no start: it starts
-- the day its countdown shipped, before scoring counted any take.
CREATE TABLE seasons (
    number    bigint      PRIMARY KEY CHECK (number >= 0 AND number <= 4294967295),
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL CHECK (ends_at > starts_at)
);

INSERT INTO seasons (number, starts_at, ends_at) VALUES (0, '2026-10-03T00:00:00Z', '2026-10-31T23:00:00Z');
