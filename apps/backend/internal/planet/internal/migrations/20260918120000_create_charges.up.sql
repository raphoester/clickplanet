-- The charges each account holds: a bomb, an enclose, a spread's clicks. They live in memory; this is what
-- a boot loads them from. A NULL time is no charge of that kind; a row holding none is deleted.
CREATE TABLE charges (
    account       uuid        PRIMARY KEY,
    bomb_until    timestamptz,
    enclose_until timestamptz,
    spread_until  timestamptz,
    spread_clicks integer     NOT NULL DEFAULT 0 CHECK (spread_clicks >= 0)
);
