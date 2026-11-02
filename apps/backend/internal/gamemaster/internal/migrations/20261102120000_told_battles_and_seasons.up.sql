-- Each battle and each season told to the other modules as it started, once.
CREATE TABLE told_battles (
    round_id text PRIMARY KEY CHECK (round_id <> '')
);

CREATE TABLE told_seasons (
    book_id text PRIMARY KEY CHECK (book_id <> '')
);

ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'gamemaster.v1.BattleStarted';
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'gamemaster.v1.SeasonStarted';
