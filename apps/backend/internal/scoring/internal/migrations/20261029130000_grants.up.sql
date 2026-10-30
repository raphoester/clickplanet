-- Points another module gave a country, beside the rounds': each counts in every book open at granted_at.
CREATE TABLE grants (
    grant_id   text        PRIMARY KEY CHECK (grant_id <> '' AND length(grant_id) <= 64),
    country    text        NOT NULL CHECK (country <> ''),
    points     integer     NOT NULL CHECK (points > 0),
    granted_at timestamptz NOT NULL
);

CREATE INDEX grants_granted_at ON grants (granted_at);
