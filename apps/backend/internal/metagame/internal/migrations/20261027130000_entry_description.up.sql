-- The entry's words for the players: what its rules are. The entries kept so far had none.
ALTER TABLE schedule_entries ADD COLUMN description text NOT NULL DEFAULT '';
