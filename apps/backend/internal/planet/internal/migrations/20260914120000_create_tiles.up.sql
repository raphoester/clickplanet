CREATE TABLE tiles (
    id      integer PRIMARY KEY CHECK (id >= 0),
    country text    NOT NULL CHECK (country <> '')
);
