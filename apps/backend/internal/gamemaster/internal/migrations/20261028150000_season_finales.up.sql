-- A season's finale, kept with its season: one row per season that has one. Its window is its length before the
-- season's end, never kept. Season 0's is the Final Battle, Sat 31 Oct 2026 21:00 to 23:00 UTC.
CREATE TABLE season_finales (
    season      bigint   PRIMARY KEY REFERENCES seasons (number) ON DELETE CASCADE,
    length      interval NOT NULL CHECK (length > interval '0'),
    rule_set    text     NOT NULL CHECK (rule_set <> ''),
    description text     NOT NULL,
    times       bigint   NOT NULL CHECK (times BETWEEN 1 AND 10)
);

INSERT INTO season_finales (season, length, rule_set, description, times)
SELECT number, interval '2 hours', 'rush', 'Triple points.', 3 FROM seasons WHERE number = 0;
