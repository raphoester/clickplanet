ALTER TABLE charges ADD COLUMN defenders integer NOT NULL DEFAULT 0 CHECK (defenders >= 0);

ALTER TABLE tiles ADD COLUMN defenders smallint NOT NULL DEFAULT 0 CHECK (defenders >= 0);
