CREATE TABLE ledger_relayed (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    position  bigint  NOT NULL CHECK (position >= 0)
);

INSERT INTO ledger_relayed (position) SELECT COALESCE(max(position) + 1, 0) FROM ledger_events;
