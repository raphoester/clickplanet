-- Every raw event of every caller. Personal data, kept for activity.retention and no longer.
CREATE TABLE events (
    -- The order written in, which the cap keeps the newest by.
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at        timestamptz NOT NULL,
    kind      text        NOT NULL,
    scope     text        NOT NULL,
    account   uuid,
    signed_in boolean     NOT NULL,

    -- What only its kind has, so a new field needs no migration. NULL for a kind with none.
    data      jsonb
);

-- The rows arrive in time order, so a few kilobytes of BRIN find a time range.
CREATE INDEX events_at ON events USING brin (at);
