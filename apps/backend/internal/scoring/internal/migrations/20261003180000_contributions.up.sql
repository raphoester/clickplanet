CREATE TABLE contributions (
    season     integer NOT NULL CHECK (season >= 0),
    account_id uuid    NOT NULL,
    country    text    NOT NULL CHECK (country <> ''),
    tiles      bigint  NOT NULL CHECK (tiles > 0),
    main       boolean NOT NULL,
    PRIMARY KEY (season, account_id, country)
);

CREATE UNIQUE INDEX contributions_main_key ON contributions (season, account_id) WHERE main;
CREATE INDEX contributions_board ON contributions (season, tiles DESC, account_id) WHERE main;
CREATE INDEX contributions_country_board ON contributions (season, country, tiles DESC, account_id) WHERE main;
CREATE INDEX contributions_account ON contributions (account_id);
