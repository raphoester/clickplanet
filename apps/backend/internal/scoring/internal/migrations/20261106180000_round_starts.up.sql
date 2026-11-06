-- What each country held when a round opened: the round ranks the tiles gained since.
CREATE TABLE round_starts (
    round_id text   NOT NULL REFERENCES rounds (round_id),
    country  text   NOT NULL CHECK (country <> ''),
    tiles    bigint NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (round_id, country)
);
