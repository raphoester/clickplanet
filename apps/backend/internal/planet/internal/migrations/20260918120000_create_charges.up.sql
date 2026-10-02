CREATE TABLE charges (
    account       uuid        PRIMARY KEY,
    bomb_until    timestamptz,
    enclose_until timestamptz,
    spread_until  timestamptz,
    spread_clicks integer     NOT NULL DEFAULT 0 CHECK (spread_clicks >= 0)
);
