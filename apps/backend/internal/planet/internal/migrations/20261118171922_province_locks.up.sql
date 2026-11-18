-- Fortify plays on provinces (docs/provinces.md): a landmass's lock goes, and the boot settles every province one flag
-- holds whole to that flag, with no shield. A lock is keyed by the provinces file it is a row of: another cut numbers
-- the provinces again.
DROP TABLE landmasses;

CREATE TABLE provinces (
    map          text     NOT NULL CHECK (map <> '' AND length(map) <= 64),
    board        smallint NOT NULL REFERENCES boards (id),
    asset        text     NOT NULL CHECK (asset <> ''),
    id           smallint NOT NULL CHECK (id > 0),
    fortified_by text     NOT NULL CHECK (fortified_by <> ''),
    PRIMARY KEY (map, asset, id)
);
