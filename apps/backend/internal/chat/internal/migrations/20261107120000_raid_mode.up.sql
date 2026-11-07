-- The one raid mode, running or ended: a second is the first replaced (single is always true).
CREATE TABLE raid_mode (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    started_at timestamptz NOT NULL,
    ends_at    timestamptz NOT NULL CHECK (ends_at >= started_at AND ends_at <= started_at + interval '24 hours'),
    reason     text        NOT NULL DEFAULT ''
);
