ALTER TABLE charges ADD COLUMN shields integer NOT NULL DEFAULT 0 CHECK (shields >= 0);

ALTER TABLE tiles ADD COLUMN shields smallint NOT NULL DEFAULT 0 CHECK (shields >= 0);
