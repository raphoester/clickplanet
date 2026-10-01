-- The flag each account and each scope takes tiles for most (clicks.Allegiance), one row per tally: each
-- country's weight as of `at`. The key is opaque: `account:<id>` or `scope:<address>`. A row with no take in
-- 3 days is deleted, so a scope is kept about as long as the ledger keeps it.
CREATE TABLE allegiances (
    key     text PRIMARY KEY,
    weights jsonb NOT NULL,
    at      timestamptz NOT NULL
);

CREATE INDEX allegiances_at ON allegiances (at);
