ALTER TABLE rounds
    ADD COLUMN samples integer NOT NULL DEFAULT 1 CHECK (samples > 0),
    ADD COLUMN map_tiles integer NOT NULL DEFAULT 1 CHECK (map_tiles > 0);

ALTER TABLE rounds ALTER COLUMN samples DROP DEFAULT, ALTER COLUMN map_tiles DROP DEFAULT;

CREATE TABLE round_holdings (
    season  integer     NOT NULL,
    ends_at timestamptz NOT NULL,
    country text        NOT NULL CHECK (country <> ''),
    tiles   bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, country),
    FOREIGN KEY (season, ends_at) REFERENCES rounds (season, ends_at)
);

ALTER TABLE round_results DROP COLUMN share;
