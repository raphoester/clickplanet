DROP INDEX contributions_country_board;
CREATE INDEX contributions_country_board ON contributions (season, country, tiles DESC, account_id) WHERE main;
