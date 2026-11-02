-- An account whose player asked to delete it is hidden from the other players; the latest decision of auth wins.
CREATE TABLE departures (
    account_id uuid        PRIMARY KEY,
    leaving    boolean     NOT NULL,
    decided_at timestamptz NOT NULL
);
