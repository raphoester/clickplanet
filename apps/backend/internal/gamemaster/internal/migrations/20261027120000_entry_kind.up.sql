-- What an entry is to the players. The entries kept so far had no look for them.
CREATE TYPE entry_kind AS ENUM ('plain', 'battle');

ALTER TABLE schedule_entries ADD COLUMN kind entry_kind NOT NULL DEFAULT 'plain';
ALTER TABLE schedule_entries ALTER COLUMN kind DROP DEFAULT;
