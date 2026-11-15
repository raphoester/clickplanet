-- BoxClaimed waits here with the claim until Redis takes it: a box caught told is a box caught paid.
CREATE TYPE outbox_event AS ENUM ('boxes.v1.BoxClaimed');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
