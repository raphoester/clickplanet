CREATE TABLE gifs (
    message_seq bigint  PRIMARY KEY REFERENCES messages (seq) ON DELETE CASCADE,
    url         text    NOT NULL,
    width       integer NOT NULL,
    height      integer NOT NULL
);
