CREATE TABLE bans (
    scope        text        PRIMARY KEY CHECK (scope <> ''),
    flags        integer     NOT NULL CHECK (flags >= 0),
    offences     integer     NOT NULL CHECK (offences >= 0),
    banned_until timestamptz NOT NULL
);

CREATE TABLE evidence (
    section  text        PRIMARY KEY CHECK (section <> ''),
    data     bytea       NOT NULL,
    saved_at timestamptz NOT NULL
);
