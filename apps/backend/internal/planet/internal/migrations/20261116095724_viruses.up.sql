-- The live viruses of each map, kept with its tiles by the same flush: a virus that ended has no row.
CREATE TABLE viruses (
    map        text        NOT NULL,
    id         uuid        NOT NULL,
    account    uuid        NOT NULL,
    scope      text        NOT NULL,
    flag       text        NOT NULL,
    victim     text        NOT NULL,
    tile       integer     NOT NULL,
    infected   integer[]   NOT NULL,
    infections integer     NOT NULL,
    next_at    timestamptz NOT NULL,
    PRIMARY KEY (map, id)
);
