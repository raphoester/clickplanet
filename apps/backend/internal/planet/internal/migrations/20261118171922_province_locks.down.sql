-- The landmasses' locks come back empty: the boot settles every landmass one flag holds whole, with no shield.
DROP TABLE provinces;

CREATE TABLE landmasses (
    asset        text     NOT NULL CHECK (asset <> ''),
    id           smallint NOT NULL CHECK (id > 0),
    fortified_by text     NOT NULL CHECK (fortified_by <> ''),
    map          text     NOT NULL CHECK (map <> '' AND length(map) <= 64),
    board        smallint NOT NULL REFERENCES boards (id),
    PRIMARY KEY (map, asset, id)
);
