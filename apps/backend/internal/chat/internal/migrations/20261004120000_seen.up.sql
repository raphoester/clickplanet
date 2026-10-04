CREATE TABLE seen (
    account_id uuid        PRIMARY KEY,
    seen_until timestamptz NOT NULL
);
