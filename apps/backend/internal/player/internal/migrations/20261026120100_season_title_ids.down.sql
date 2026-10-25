DELETE FROM worn_titles WHERE title !~ '^[a-z_]+$';
DELETE FROM titles WHERE title !~ '^[a-z_]+$';
ALTER TABLE titles DROP CONSTRAINT titles_title_check, ADD CONSTRAINT titles_title_check CHECK (title ~ '^[a-z_]+$');
ALTER TABLE worn_titles DROP CONSTRAINT worn_titles_title_check, ADD CONSTRAINT worn_titles_title_check CHECK (title ~ '^[a-z_]+$');
