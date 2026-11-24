-- The time each payment granted, named by the payment: the shop tells a payment again, a grant is kept once. A refund
-- takes back what it added.
CREATE TABLE time_grants (
    id         text        PRIMARY KEY,
    -- The flag the time went to: a submission's, once the payment opened it.
    flag       text        NOT NULL REFERENCES flags (id),
    -- How much later the flag ended for it, or how long it opened for.
    added      interval    NOT NULL,
    granted_at timestamptz NOT NULL
);

-- A payment refunded or lost in a dispute. One taken back before its grant was heard grants nothing.
CREATE TABLE time_takebacks (
    grant_id   text        PRIMARY KEY,
    decided_at timestamptz NOT NULL
);
