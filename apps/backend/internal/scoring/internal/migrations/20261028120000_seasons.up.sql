-- Each season as the gamemaster told it last. Season 0 was the YAML's calendar, with no start: it starts the day its
-- countdown shipped, before any take was counted. Decided at its start, every command of the gamemaster overrules it.
CREATE TABLE seasons (
    number     bigint      PRIMARY KEY CHECK (number >= 0 AND number <= 4294967295),
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz NOT NULL CHECK (ends_at > starts_at),
    decided_at timestamptz NOT NULL
);

INSERT INTO seasons (number, starts_at, ends_at, decided_at)
VALUES (0, '2026-10-03T00:00:00Z', '2026-10-31T23:00:00Z', '2026-10-03T00:00:00Z');
