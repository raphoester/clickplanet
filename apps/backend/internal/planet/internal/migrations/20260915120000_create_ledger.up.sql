CREATE TABLE ledger_takes (
    position bigint      PRIMARY KEY CHECK (position >= 0),
    tile     integer     NOT NULL CHECK (tile >= 0),
    scope    text        NOT NULL,
    country  text        NOT NULL,
    previous text        NOT NULL,
    taken_at timestamptz NOT NULL
);

CREATE TABLE ledger_head (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    head      bigint  NOT NULL CHECK (head >= 0)
);

CREATE TABLE ledger_forgotten (
    scope           text   PRIMARY KEY,
    before_position bigint NOT NULL CHECK (before_position >= 0)
);
