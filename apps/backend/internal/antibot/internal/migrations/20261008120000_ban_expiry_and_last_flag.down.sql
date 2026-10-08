ALTER TABLE account_bans DROP COLUMN last_flagged_at;
ALTER TABLE account_bans RENAME COLUMN expires_at TO banned_until;

ALTER TABLE bans DROP COLUMN last_flagged_at;
ALTER TABLE bans RENAME COLUMN expires_at TO banned_until;
