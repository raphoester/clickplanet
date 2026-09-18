-- What the chat says on its own, between the messages: a bomb that landed. No sender, so nothing personal, but it
-- is kept for chat.storage.retention with the messages it sits between.
-- kind says how to read payload, the values the client writes the line from. The server makes the id, so unlike a
-- message's it is a real key. A read orders by time, then id.
CREATE TABLE announcements (
    id           uuid        PRIMARY KEY,
    kind         text        NOT NULL,
    payload      jsonb       NOT NULL,
    announced_at timestamptz NOT NULL
);

-- The prune deletes by age.
CREATE INDEX announcements_announced_at ON announcements (announced_at);
