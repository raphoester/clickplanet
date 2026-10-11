ALTER TABLE messages ADD COLUMN reply_to text;

CREATE INDEX messages_id ON messages (id);
