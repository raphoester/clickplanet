-- The capitals an arena's seed gave the flags with no ground (England, Scotland, Wales, the flags sold), on the arena's
-- board, from each flag's pin: kept with the map's tiles, deleted by its wipe.
CREATE TABLE seats (
    map   text     NOT NULL,
    board smallint NOT NULL REFERENCES boards (id),
    flag  text     NOT NULL,
    tile  integer  NOT NULL,
    PRIMARY KEY (map, flag),
    UNIQUE (map, tile)
);
