-- The UTC days the operator forgave an account's streak, by progression.v1.AdminService/ForgiveMissedDay: on the
-- streak, never a day played, so the days played backfill counts none of them.
ALTER TABLE stats ADD COLUMN IF NOT EXISTS forgiven_days date[] NOT NULL DEFAULT '{}';
