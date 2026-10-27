DROP TABLE ended_seasons;
DROP TABLE round_players;
DROP TABLE round_takes;

DELETE FROM round_results WHERE own;
DELETE FROM rounds WHERE own;

ALTER TABLE round_results DROP CONSTRAINT round_results_season_ends_at_own_fkey;
ALTER TABLE round_results DROP CONSTRAINT round_results_pkey;
ALTER TABLE rounds DROP CONSTRAINT rounds_pkey;
ALTER TABLE rounds ADD PRIMARY KEY (season, ends_at);
ALTER TABLE round_results ADD PRIMARY KEY (season, ends_at, country);
ALTER TABLE round_results ADD FOREIGN KEY (season, ends_at) REFERENCES rounds (season, ends_at);
ALTER TABLE round_results DROP COLUMN own;

ALTER TABLE rounds
    DROP CONSTRAINT rounds_own_starts,
    DROP COLUMN counted_takes_below,
    DROP COLUMN times,
    DROP COLUMN starts_at,
    DROP COLUMN own;
