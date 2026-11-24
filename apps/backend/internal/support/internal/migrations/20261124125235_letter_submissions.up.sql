-- A letter is about one flag submission, and a submission has one letter at most. The letters written before had no
-- submission to name: they go, with their mails, replies and closings.
DELETE FROM letters;

ALTER TABLE letters ADD COLUMN submission_id uuid NOT NULL;

CREATE UNIQUE INDEX letters_submission ON letters (submission_id);
