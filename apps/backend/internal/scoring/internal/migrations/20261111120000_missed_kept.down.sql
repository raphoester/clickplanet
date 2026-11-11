CREATE TYPE outbox_event AS ENUM (
    'scoring.v1.RoundScored', 'scoring.v1.BookClosed', 'scoring.v1.RoundMissed', 'scoring.v1.BookMissed'
);

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);

DROP TABLE missed_books, missed_rounds;
