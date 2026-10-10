CREATE TABLE landmasses (
    asset        text     NOT NULL CHECK (asset <> ''),
    id           smallint NOT NULL CHECK (id > 0),
    fortified_by text     NOT NULL CHECK (fortified_by <> ''),
    PRIMARY KEY (asset, id)
);
