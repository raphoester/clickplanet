CREATE TABLE mutes (
    id          uuid        PRIMARY KEY,
    account_id  uuid        NOT NULL,
    scope       text,
    muted_at    timestamptz NOT NULL,
    muted_until timestamptz NOT NULL
);

CREATE INDEX mutes_account ON mutes (account_id, muted_until);

CREATE INDEX mutes_scope ON mutes (scope, muted_until) WHERE scope IS NOT NULL;

CREATE INDEX mutes_muted_until ON mutes (muted_until);
