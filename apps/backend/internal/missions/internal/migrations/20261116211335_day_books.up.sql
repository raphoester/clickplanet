-- The book of each day's season, which scoring ranks the flags in: a flag mission's points are drawn from its table.
-- Every day id so far is the gamemaster's round id, season-N/day-K, and its book is season-N.
ALTER TABLE mission_days ADD COLUMN book_id text;

UPDATE mission_days SET book_id = substring(day_id FROM '^(season-[0-9]+)/day-[0-9]+$');

ALTER TABLE mission_days
    ALTER COLUMN book_id SET NOT NULL,
    ADD CHECK (book_id <> '');
