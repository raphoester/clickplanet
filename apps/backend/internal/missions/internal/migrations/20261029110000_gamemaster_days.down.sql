DELETE FROM mission_sheets WHERE day !~ '^season-0/day-\d+$';
DELETE FROM mission_briefs WHERE day !~ '^season-0/day-\d+$';

ALTER TABLE mission_progress DROP CONSTRAINT mission_progress_day_account_id_fkey;
ALTER TABLE mission_payments DROP CONSTRAINT mission_payments_day_account_id_fkey;

ALTER TABLE mission_briefs ALTER COLUMN day TYPE date USING date '2026-10-07' + substring(day from '\d+$')::integer;
ALTER TABLE mission_sheets ALTER COLUMN day TYPE date USING date '2026-10-07' + substring(day from '\d+$')::integer;
ALTER TABLE mission_progress ALTER COLUMN day TYPE date USING date '2026-10-07' + substring(day from '\d+$')::integer;
ALTER TABLE mission_payments ALTER COLUMN day TYPE date USING date '2026-10-07' + substring(day from '\d+$')::integer;

ALTER TABLE mission_progress ADD FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE;
ALTER TABLE mission_payments ADD FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE;

DROP TABLE mission_days;
