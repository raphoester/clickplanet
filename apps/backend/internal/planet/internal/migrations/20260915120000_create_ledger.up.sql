-- The ledger: every take of every tile, oldest first. It lives in memory; this is what a boot loads it from.
-- Takes are only appended, and deleted from the front as the retention or the cap drops them.
CREATE TABLE ledger_takes (
    position bigint      PRIMARY KEY CHECK (position >= 0),
    tile     integer     NOT NULL CHECK (tile >= 0),
    scope    text        NOT NULL,
    country  text        NOT NULL,
    previous text        NOT NULL,
    taken_at timestamptz NOT NULL
);

-- The oldest position kept. One row, so positions carry on past a ledger the retention emptied.
CREATE TABLE ledger_head (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    head      bigint  NOT NULL CHECK (head >= 0)
);

-- A reverted scope's takes before before_position are forgotten.
CREATE TABLE ledger_forgotten (
    scope           text   PRIMARY KEY,
    before_position bigint NOT NULL CHECK (before_position >= 0)
);
