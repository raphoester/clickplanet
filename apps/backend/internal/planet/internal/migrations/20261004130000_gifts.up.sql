CREATE TABLE gifts (
    tag      text        NOT NULL,
    account  uuid        NOT NULL,
    given_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tag, account)
);
