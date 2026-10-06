ALTER TABLE charges ADD COLUMN defenders integer NOT NULL DEFAULT 0 CHECK (defenders >= 0);

CREATE TABLE garrisons (
    tile      integer PRIMARY KEY CHECK (tile > 0),
    country   text    NOT NULL CHECK (country <> ''),
    defenders integer NOT NULL CHECK (defenders > 0)
);
