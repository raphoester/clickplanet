-- The mines laid on each map, kept with its tiles by the same flush: one a tile at most, its flag the tile's.
CREATE TABLE mines (
    map     text    NOT NULL,
    tile    integer NOT NULL,
    flag    text    NOT NULL,
    account uuid    NOT NULL,
    scope   text    NOT NULL,
    PRIMARY KEY (map, tile)
);
