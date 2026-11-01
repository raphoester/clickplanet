CREATE TABLE podium_places (
    season     integer NOT NULL CHECK (season >= 0),
    account_id uuid    NOT NULL,
    rank       integer NOT NULL CHECK (rank BETWEEN 1 AND 3),
    PRIMARY KEY (season, account_id)
);

CREATE INDEX podium_places_account ON podium_places (account_id);
