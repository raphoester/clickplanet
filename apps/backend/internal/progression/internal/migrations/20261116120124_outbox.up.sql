-- What a level unlocked waits here with the level until Redis takes it: a level kept is a level told to the bonuses.
CREATE TYPE outbox_event AS ENUM ('bonuses.v1.KeepUnlocks');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
