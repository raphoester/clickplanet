-- Every raw event of every caller. Personal data, kept for activity.retention and no longer.
CREATE TABLE events (
    -- The order written in, which the cap keeps the newest by.
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at        timestamptz NOT NULL,
    kind      text        NOT NULL,
    scope     text        NOT NULL,
    account   uuid,
    signed_in boolean     NOT NULL,

    -- click and take. bigint: a refused click may name any uint32.
    tile      bigint,
    country   text,

    -- click and map.
    outcome   text,

    -- take: who held the tile before, NULL for nobody.
    held      text,

    -- map, as asked: map_end 0 is the end of the map.
    map_start bigint,
    map_end   bigint,
    off_map   boolean,

    -- box_caught.
    delay_us  bigint
);

-- The rows arrive in time order, so a few kilobytes of BRIN find a time range.
CREATE INDEX events_at ON events USING brin (at);
