DELETE FROM mission_sheets;
ALTER TABLE mission_sheets ADD COLUMN flag text NOT NULL CHECK (flag <> '');

DROP TABLE mission_briefs;

CREATE TABLE mission_briefs (
    day      date  NOT NULL,
    flag     text  NOT NULL CHECK (flag <> ''),
    missions jsonb NOT NULL CHECK (jsonb_typeof(missions) = 'array'),
    PRIMARY KEY (day, flag)
);
