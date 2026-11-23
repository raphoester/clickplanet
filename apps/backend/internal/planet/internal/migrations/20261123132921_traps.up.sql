-- The traps laid on each map, kept with its tiles by the same flush: one a tile at most, beside a mine or not, its flag
-- the tile's.
CREATE TABLE traps (
    map     text     NOT NULL,
    board   smallint NOT NULL REFERENCES boards (id),
    tile    integer  NOT NULL,
    flag    text     NOT NULL,
    account uuid     NOT NULL,
    PRIMARY KEY (map, tile)
);
