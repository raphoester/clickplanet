ALTER TABLE profiles ADD COLUMN color smallint NOT NULL DEFAULT 0 CHECK (color >= 0);
