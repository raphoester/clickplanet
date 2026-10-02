-- Which account sent the message, so a reader is shown who its player is now rather than a copy of the name it
-- went by then. A rename then shows on everything that player ever said, and a deleted account stops being
-- named at all — the copy in `name` could do neither.
--
-- Nullable because a row written before this has no account to fill in: those keep `name` and `author_admin`,
-- and the read path falls back to them. See the TODO in postgres_message_store: once chat.storage.retention has
-- passed, every remaining row has an account, and the two old columns and the fallback can go.
ALTER TABLE messages ADD COLUMN account_id uuid;

-- The history resolves the accounts of a whole window at once; this keeps that read off the name columns.
CREATE INDEX messages_account_id ON messages (account_id);
