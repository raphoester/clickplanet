CREATE TYPE outbox_event AS ENUM ('player.v1.PlayerCreated');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
