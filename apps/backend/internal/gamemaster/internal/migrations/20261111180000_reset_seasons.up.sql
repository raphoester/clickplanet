-- Each season whose map planet answered reset, by its book: the reset is asked no more.
CREATE TABLE reset_seasons (
    book_id text PRIMARY KEY CHECK (book_id <> '')
);
