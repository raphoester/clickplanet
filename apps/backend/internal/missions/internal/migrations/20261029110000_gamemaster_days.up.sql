-- Each day the gamemaster told: a mission row names its day by the day's round id.
CREATE TABLE mission_days (
    day_id    text        PRIMARY KEY CHECK (day_id <> ''),
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL CHECK (ends_at > starts_at)
);

ALTER TABLE mission_progress DROP CONSTRAINT mission_progress_day_account_id_fkey;
ALTER TABLE mission_payments DROP CONSTRAINT mission_payments_day_account_id_fkey;

-- Every row so far is of season 0, whose day K ended at 21:00 UTC, K days after 7 Oct 2026: the date it ended on.
ALTER TABLE mission_briefs ALTER COLUMN day TYPE text USING 'season-0/day-' || (day - date '2026-10-07');
ALTER TABLE mission_sheets ALTER COLUMN day TYPE text USING 'season-0/day-' || (day - date '2026-10-07');
ALTER TABLE mission_progress ALTER COLUMN day TYPE text USING 'season-0/day-' || (day - date '2026-10-07');
ALTER TABLE mission_payments ALTER COLUMN day TYPE text USING 'season-0/day-' || (day - date '2026-10-07');

ALTER TABLE mission_progress ADD FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE;
ALTER TABLE mission_payments ADD FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE;
