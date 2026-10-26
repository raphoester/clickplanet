CREATE TYPE outbox_event AS ENUM ('planet.v1.RefuseCaller', 'planet.v1.AdmitCaller');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
