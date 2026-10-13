DELETE FROM notes WHERE kind = 'bug';

ALTER TABLE notes DROP COLUMN diagnostics;
