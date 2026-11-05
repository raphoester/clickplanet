CREATE TYPE outbox_event AS ENUM ('seasons.v1.RoundClosed', 'seasons.v1.SeasonClosed');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
