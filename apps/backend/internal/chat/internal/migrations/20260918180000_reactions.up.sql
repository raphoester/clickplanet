-- Who put which reaction on which message: one row per reactor and reaction. A reactor is a player's account or
-- a guest's tag, so this is personal data kept for chat.storage.retention, like the messages.
-- reaction is chat.v1.Reaction's number, which the proto never reuses.
CREATE TABLE reactions (
    message_id text        NOT NULL,
    reaction   smallint    NOT NULL,
    reactor    text        NOT NULL,
    reacted_at timestamptz NOT NULL,
    PRIMARY KEY (message_id, reaction, reactor)
);

-- The prune deletes by age.
CREATE INDEX reactions_reacted_at ON reactions (reacted_at);
