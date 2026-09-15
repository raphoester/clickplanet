-- One row per scope ever banned. Offences are never forgotten, so a row is never deleted by the process.
CREATE TABLE bans (
    scope        text        PRIMARY KEY CHECK (scope <> ''),
    flags        integer     NOT NULL CHECK (flags >= 0),
    offences     integer     NOT NULL CHECK (offences >= 0),
    banned_until timestamptz NOT NULL
);

-- One row per watchdog, and one for the jury: what it tracks, encoded by its own package.
-- Every flush replaces every row, so nothing older than the retention outlives one flush.
CREATE TABLE evidence (
    section  text        PRIMARY KEY CHECK (section <> ''),
    data     bytea       NOT NULL,
    saved_at timestamptz NOT NULL
);
