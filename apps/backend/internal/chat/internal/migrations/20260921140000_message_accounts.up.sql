ALTER TABLE messages ADD COLUMN account_id uuid;

CREATE INDEX messages_account_id ON messages (account_id);
