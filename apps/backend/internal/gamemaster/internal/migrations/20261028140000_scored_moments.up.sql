-- What scoring scored, as it told it: each round and each book once. A new one is told in the game's words through the outbox.
CREATE TABLE scored_rounds (
    round_id text        PRIMARY KEY CHECK (round_id <> ''),
    book_id  text        NOT NULL CHECK (book_id <> ''),
    ended_at timestamptz NOT NULL
);

-- Each round the gamemaster opened: a day that opened runs to the season's end when the end moves before its cutoff.
CREATE TABLE opened_rounds (
    round_id text PRIMARY KEY CHECK (round_id <> '')
);

CREATE TABLE closed_books (
    book_id  text        PRIMARY KEY CHECK (book_id <> ''),
    ended_at timestamptz NOT NULL
);

CREATE TYPE outbox_event AS ENUM ('gamemaster.v1.DayEnded', 'gamemaster.v1.BattleEnded', 'gamemaster.v1.SeasonEnded');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
