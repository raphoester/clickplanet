CREATE TABLE messages (
    seq        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id         text        NOT NULL,
    sent_at    timestamptz NOT NULL,
    name       text        NOT NULL,
    tag        text        NOT NULL,
    author_id  text        NOT NULL,
    country    text        NOT NULL,
    ip         text        NOT NULL,
    user_agent text        NOT NULL,
    text       text        NOT NULL
);

CREATE INDEX messages_sent_at ON messages (sent_at);
