CREATE TYPE outbox_event AS ENUM ('auth.v1.AccountDeleted', 'auth.v1.SignedIn');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
