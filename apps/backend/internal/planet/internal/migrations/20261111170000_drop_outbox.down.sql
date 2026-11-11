CREATE TYPE outbox_event AS ENUM ('planet.v1.ArenaClosed', 'planet.v1.ArenaOpened');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
