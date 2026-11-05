ALTER TABLE round_results ADD COLUMN share double precision NOT NULL DEFAULT 0;

UPDATE round_results AS result SET share = coalesce((
    SELECT held.tiles FROM round_holdings AS held
    WHERE held.season = result.season AND held.ends_at = result.ends_at AND held.country = result.country
), 0)::double precision / (
    SELECT round.samples::double precision * round.map_tiles FROM rounds AS round
    WHERE round.season = result.season AND round.ends_at = result.ends_at
);

ALTER TABLE round_results
    ALTER COLUMN share DROP DEFAULT,
    ADD CONSTRAINT round_results_share CHECK (share >= 0 AND share <= 1);

DROP TABLE round_holdings;

ALTER TABLE rounds DROP COLUMN samples, DROP COLUMN map_tiles;
