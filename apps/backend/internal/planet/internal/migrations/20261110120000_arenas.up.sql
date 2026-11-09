-- One table keeps every map's tiles and fortify locks: the world, the game's own map, and the blank map of each arena
-- open. A column added to a tile, or a remap of the ids, reaches every map. The tile map plays one map at a time.
ALTER TABLE tiles ADD COLUMN map text NOT NULL DEFAULT 'world' CHECK (map <> '' AND length(map) <= 64);
ALTER TABLE tiles ALTER COLUMN map DROP DEFAULT;
ALTER TABLE tiles DROP CONSTRAINT tiles_pkey, ADD PRIMARY KEY (map, id);

ALTER TABLE landmasses ADD COLUMN map text NOT NULL DEFAULT 'world' CHECK (map <> '' AND length(map) <= 64);
ALTER TABLE landmasses ALTER COLUMN map DROP DEFAULT;
ALTER TABLE landmasses DROP CONSTRAINT landmasses_pkey, ADD PRIMARY KEY (map, asset, id);

-- An arena is the blank map an arena battle is fought on. A row once its map plays; its tiles are the map named by its
-- id until it closes.
CREATE TABLE arenas (
    id        text        PRIMARY KEY CHECK (id <> '' AND length(id) <= 64 AND id <> 'world'),
    ends_at   timestamptz NOT NULL,
    opened_at timestamptz NOT NULL
);

-- Each arena closed: its map deleted and the world's played again, or it never opened.
CREATE TABLE closed_arenas (
    id        text        PRIMARY KEY CHECK (id <> '' AND length(id) <= 64),
    closed_at timestamptz NOT NULL
);

CREATE TYPE outbox_event AS ENUM ('planet.v1.ArenaClosed');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
