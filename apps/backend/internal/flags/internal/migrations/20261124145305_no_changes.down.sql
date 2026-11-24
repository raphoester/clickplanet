CREATE TYPE submission_kind AS ENUM ('new', 'change');

ALTER TABLE submissions ADD COLUMN kind submission_kind NOT NULL DEFAULT 'new';

ALTER TABLE submissions ALTER COLUMN kind DROP DEFAULT;
