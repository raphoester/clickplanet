-- Each round scoring told missed: closed before it ever opened, so nothing scores it and nothing waits on it.
CREATE TABLE missed_rounds (
    round_id text        PRIMARY KEY CHECK (round_id <> ''),
    ended_at timestamptz NOT NULL
);

-- Each book scoring told missed: closed before it ever opened, so nothing ranks it.
CREATE TABLE missed_books (
    book_id  text        PRIMARY KEY CHECK (book_id <> ''),
    ended_at timestamptz NOT NULL
);
