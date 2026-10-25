CREATE TABLE seasons_played (
    account_id uuid    NOT NULL,
    season     integer NOT NULL CHECK (season >= 0),
    country    text    NOT NULL,
    won        boolean NOT NULL,
    PRIMARY KEY (account_id, season)
);
