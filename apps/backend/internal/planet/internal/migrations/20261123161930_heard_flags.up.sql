-- The flags opened since the game shipped, as this module heard them (flags.v1.FlagOpened, FlagClosed): the latest
-- fact of each holds. The flags the game ships with are in the code.
CREATE TYPE heard_flag_state AS ENUM ('open', 'closed');

CREATE TABLE heard_flags (
    flag       text             PRIMARY KEY,
    state      heard_flag_state NOT NULL,
    decided_at timestamptz      NOT NULL
);
