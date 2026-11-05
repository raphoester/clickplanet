-- A round of its own is the window of a schedule entry whose rule set asked for one: kept beside the days, with its
-- start and its points' times. A day and a round of its own may end at one time, so the kind is in the key.
ALTER TABLE rounds
    ADD COLUMN own                 boolean     NOT NULL DEFAULT false,
    ADD COLUMN starts_at           timestamptz,
    ADD COLUMN times               integer     NOT NULL DEFAULT 1 CHECK (times BETWEEN 1 AND 10),
    ADD COLUMN counted_takes_below bigint      NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0),
    ADD CONSTRAINT rounds_own_starts CHECK (own = (starts_at IS NOT NULL));

ALTER TABLE round_results ADD COLUMN own boolean NOT NULL DEFAULT false;

ALTER TABLE round_results DROP CONSTRAINT round_results_season_ends_at_fkey;
ALTER TABLE round_results DROP CONSTRAINT round_results_pkey;
ALTER TABLE rounds DROP CONSTRAINT rounds_pkey;
ALTER TABLE rounds ADD PRIMARY KEY (season, ends_at, own);
ALTER TABLE round_results ADD PRIMARY KEY (season, ends_at, own, country);
ALTER TABLE round_results ADD FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own);

ALTER TABLE rounds ALTER COLUMN own DROP DEFAULT, ALTER COLUMN times DROP DEFAULT;
ALTER TABLE round_results ALTER COLUMN own DROP DEFAULT;

-- The tiles each flag's players took during a round of its own, and each account's, for its numbers.
CREATE TABLE round_takes (
    season  integer     NOT NULL,
    ends_at timestamptz NOT NULL,
    own     boolean     NOT NULL CHECK (own),
    country text        NOT NULL CHECK (country <> ''),
    tiles   bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, own, country),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

CREATE TABLE round_players (
    season     integer     NOT NULL,
    ends_at    timestamptz NOT NULL,
    own        boolean     NOT NULL CHECK (own),
    account_id uuid        NOT NULL,
    country    text        NOT NULL CHECK (country <> ''),
    tiles      bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, own, account_id, country),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

CREATE INDEX round_players_account ON round_players (account_id);

-- A season ends at its end, whether a round ends with it or none does: once ended, it is told once.
CREATE TABLE ended_seasons (
    season   integer     PRIMARY KEY CHECK (season >= 0),
    ended_at timestamptz NOT NULL
);

INSERT INTO ended_seasons (season, ended_at) SELECT season, ends_at FROM rounds WHERE finale AND closed;
