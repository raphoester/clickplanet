-- The flags opened since the game shipped; the ones it ships with are in the code. A flag is closed, never deleted:
-- history still names it.
CREATE TYPE flag_state AS ENUM ('open', 'closed');

CREATE TABLE flags (
    id         text        PRIMARY KEY,
    name       text        NOT NULL,
    state      flag_state  NOT NULL,
    -- When it took its state.
    decided_at timestamptz NOT NULL
);

-- What the flags module tells the other modules, kept in the transaction of what it tells and sent by the relay.
CREATE TYPE outbox_event AS ENUM ('flags.v1.FlagOpened', 'flags.v1.FlagClosed');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
