-- What support tells the other modules, kept in the transaction of what it tells and sent by the relay.
CREATE TYPE outbox_event AS ENUM ('support.v1.PlayerReported');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
