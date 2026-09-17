-- One row per account ever banned, beside bans on scopes. Offences are never forgotten, so a row is never deleted by the process.
CREATE TABLE account_bans (
    account      uuid        PRIMARY KEY,
    flags        integer     NOT NULL CHECK (flags >= 0),
    offences     integer     NOT NULL CHECK (offences >= 0),
    banned_until timestamptz NOT NULL
);
