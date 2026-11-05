ALTER TABLE titles DROP CONSTRAINT titles_title_check;
ALTER TABLE titles ADD CONSTRAINT titles_title_check CHECK (title ~ '^[a-z0-9_]+$');
ALTER TABLE worn_titles DROP CONSTRAINT worn_titles_title_check;
ALTER TABLE worn_titles ADD CONSTRAINT worn_titles_title_check CHECK (title ~ '^[a-z0-9_]+$');
