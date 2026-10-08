CREATE TABLE fronts (
    account_id    uuid   NOT NULL,
    country       text   NOT NULL CHECK (country <> ''),
    plays_for     bigint NOT NULL DEFAULT 0 CHECK (plays_for >= 0),
    plays_against bigint NOT NULL DEFAULT 0 CHECK (plays_against >= 0),
    PRIMARY KEY (account_id, country)
);

-- The seasons counted every take for its flag since they shipped, so their rows are where a player's total starts.
DO $$
BEGIN
    IF to_regclass('seasons.contributions') IS NOT NULL THEN
        INSERT INTO fronts (account_id, country, plays_for)
        SELECT account_id, country, sum(tiles) FROM seasons.contributions GROUP BY account_id, country;
    END IF;
END
$$;
