CREATE TABLE rounds (
    season    integer     NOT NULL CHECK (season >= 0),
    ends_at   timestamptz NOT NULL,
    finale    boolean     NOT NULL,
    samples   integer     NOT NULL CHECK (samples > 0),
    map_tiles integer     NOT NULL CHECK (map_tiles > 0),
    closed    boolean     NOT NULL DEFAULT false,
    PRIMARY KEY (season, ends_at)
);

CREATE INDEX rounds_unclosed ON rounds (ends_at) WHERE NOT closed;

CREATE TABLE round_holdings (
    season  integer     NOT NULL,
    ends_at timestamptz NOT NULL,
    country text        NOT NULL CHECK (country <> ''),
    tiles   bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, country),
    FOREIGN KEY (season, ends_at) REFERENCES rounds (season, ends_at)
);

CREATE TABLE round_results (
    season  integer     NOT NULL,
    ends_at timestamptz NOT NULL,
    country text        NOT NULL CHECK (country <> ''),
    rank    integer     NOT NULL CHECK (rank > 0),
    points  integer     NOT NULL CHECK (points >= 0),
    PRIMARY KEY (season, ends_at, country),
    FOREIGN KEY (season, ends_at) REFERENCES rounds (season, ends_at)
);
