-- The tags dropped do not come back.
ALTER TABLE messages ADD COLUMN tag text NOT NULL DEFAULT '';
