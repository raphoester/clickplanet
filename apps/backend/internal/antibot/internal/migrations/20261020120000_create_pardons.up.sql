CREATE TABLE pardons (
    scope        text        NOT NULL REFERENCES bans (scope),
    pardoned_at  timestamptz NOT NULL,
    flags        integer     NOT NULL CHECK (flags >= 0),
    offence      integer     NOT NULL CHECK (offence >= 0),
    banned_until timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    PRIMARY KEY (scope, pardoned_at)
);

CREATE TABLE account_pardons (
    account      uuid        NOT NULL REFERENCES account_bans (account),
    pardoned_at  timestamptz NOT NULL,
    flags        integer     NOT NULL CHECK (flags >= 0),
    offence      integer     NOT NULL CHECK (offence >= 0),
    banned_until timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    PRIMARY KEY (account, pardoned_at)
);
