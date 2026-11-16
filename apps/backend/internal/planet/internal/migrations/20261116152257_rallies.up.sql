-- The last rally each account called, on or ended: one a player, so a call replaces the one before once it ended.
CREATE TABLE rallies (
    account   uuid             PRIMARY KEY,
    id        uuid             NOT NULL,
    flag      text             NOT NULL CHECK (flag <> ''),
    tile      integer          NOT NULL,
    radius    double precision NOT NULL,
    called_at timestamptz      NOT NULL,
    ends_at   timestamptz      NOT NULL
);

CREATE INDEX rallies_flag_ends_at ON rallies (flag, ends_at);
