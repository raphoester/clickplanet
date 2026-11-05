CREATE TABLE finale_fights (
    season     integer NOT NULL CHECK (season >= 0),
    account_id uuid    NOT NULL,
    country    text    NOT NULL CHECK (country <> ''),
    -- 0: the flag is not on the season's table.
    flag_rank  integer NOT NULL CHECK (flag_rank >= 0),
    PRIMARY KEY (season, account_id)
);

CREATE INDEX finale_fights_account ON finale_fights (account_id);
