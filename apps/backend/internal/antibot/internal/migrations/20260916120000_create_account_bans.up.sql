CREATE TABLE account_bans (
    account      uuid        PRIMARY KEY,
    flags        integer     NOT NULL CHECK (flags >= 0),
    offences     integer     NOT NULL CHECK (offences >= 0),
    banned_until timestamptz NOT NULL
);
