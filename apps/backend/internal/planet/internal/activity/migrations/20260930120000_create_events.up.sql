-- Every raw event of every caller. Personal data, kept for activity.retention and no longer.
CREATE TABLE events (
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at        timestamptz NOT NULL,
    kind      text        NOT NULL,
    scope     text        NOT NULL,
    account   uuid,
    signed_in boolean     NOT NULL,
    data      jsonb
);

CREATE INDEX events_at ON events USING brin (at);
