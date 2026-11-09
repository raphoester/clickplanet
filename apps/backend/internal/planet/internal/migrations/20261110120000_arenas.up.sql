-- An arena is the blank map of a battle played as a siege. A row once the map went blank, with the map from before it
-- in arena_tiles and arena_landmasses until it closes.
CREATE TABLE arenas (
    id        text        PRIMARY KEY CHECK (id <> '' AND length(id) <= 64),
    ends_at   timestamptz NOT NULL,
    opened_at timestamptz NOT NULL
);

CREATE TABLE arena_tiles (
    arena   text     NOT NULL REFERENCES arenas (id),
    id      integer  NOT NULL CHECK (id >= 0),
    country text     NOT NULL CHECK (country <> ''),
    shields smallint NOT NULL CHECK (shields >= 0),
    holder  uuid,
    PRIMARY KEY (arena, id)
);

CREATE INDEX arena_tiles_holder ON arena_tiles (holder) WHERE holder IS NOT NULL;

CREATE TABLE arena_landmasses (
    arena        text     NOT NULL REFERENCES arenas (id),
    asset        text     NOT NULL CHECK (asset <> ''),
    id           smallint NOT NULL CHECK (id > 0),
    fortified_by text     NOT NULL CHECK (fortified_by <> ''),
    PRIMARY KEY (arena, asset, id)
);

-- Each arena closed: the map from before it brought back, or it never opened.
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
