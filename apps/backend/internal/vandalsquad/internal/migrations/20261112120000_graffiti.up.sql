CREATE TYPE graffiti_kind AS ENUM ('swastika');

CREATE TABLE graffiti (
    id         uuid          PRIMARY KEY,
    -- NULL once the painter's account is forgotten.
    account    uuid,
    flag       text          NOT NULL,
    kind       graffiti_kind NOT NULL,
    tiles      integer[]     NOT NULL,
    flagged_at timestamptz   NOT NULL,
    seen_at    timestamptz   NOT NULL,
    link       text          NOT NULL
);

CREATE INDEX graffiti_account ON graffiti (account) WHERE account IS NOT NULL;

CREATE INDEX graffiti_flagged_at ON graffiti (flagged_at);
