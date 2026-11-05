-- Each day told to the other modules, with the end it was told with: a moved season end tells it again.
CREATE TABLE told_days (
    round_id text        PRIMARY KEY CHECK (round_id <> ''),
    ends_at  timestamptz NOT NULL
);

ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'gamemaster.v1.DayStarted';
