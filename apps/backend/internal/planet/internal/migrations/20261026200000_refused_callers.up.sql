CREATE TYPE caller_kind AS ENUM ('scope', 'account');

CREATE TABLE refused_callers (
    kind          caller_kind NOT NULL,
    caller        text        NOT NULL CHECK (caller <> ''),
    refused_until timestamptz NOT NULL,
    decided_at    timestamptz NOT NULL,
    PRIMARY KEY (kind, caller)
);
