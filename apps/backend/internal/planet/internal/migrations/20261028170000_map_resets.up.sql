-- Each map reset planet applied, kept once by its id, so a reset told again wipes nothing. at is when it was due: a
-- replay never reaches before it.
CREATE TABLE map_resets (
    id text        PRIMARY KEY CHECK (id <> '' AND length(id) <= 64),
    at timestamptz NOT NULL
);
