CREATE TABLE reactions (
    message_id text        NOT NULL,
    reaction   smallint    NOT NULL,
    reactor    text        NOT NULL,
    reacted_at timestamptz NOT NULL,
    PRIMARY KEY (message_id, reaction, reactor)
);

CREATE INDEX reactions_reacted_at ON reactions (reacted_at);
