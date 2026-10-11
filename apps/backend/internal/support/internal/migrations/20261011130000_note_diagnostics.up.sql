ALTER TYPE note_kind ADD VALUE IF NOT EXISTS 'bug';

ALTER TABLE notes ADD COLUMN diagnostics jsonb;
