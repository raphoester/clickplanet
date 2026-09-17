-- Every chat message, sender IP included: the audit trail, and personal data kept only for chat.storage.retention.
-- seq is the order messages were accepted in; id is the message's own id and is not trusted to be unique.
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

-- The prune deletes by age.
CREATE INDEX messages_sent_at ON messages (sent_at);
