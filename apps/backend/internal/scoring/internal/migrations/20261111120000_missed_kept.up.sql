-- A round or a book closed before it ever opened: it never opens, and every close of it answers this end.
CREATE TABLE missed_rounds (
    round_id text        PRIMARY KEY CHECK (round_id <> '' AND length(round_id) <= 64),
    ended_at timestamptz NOT NULL
);

CREATE TABLE missed_books (
    book_id  text        PRIMARY KEY CHECK (book_id <> '' AND length(book_id) <= 64),
    ended_at timestamptz NOT NULL
);

-- Each close is answered over the gamemaster's call: no event is left to relay.
DELETE FROM outbox;
DROP TABLE outbox;
DROP TYPE outbox_event;
