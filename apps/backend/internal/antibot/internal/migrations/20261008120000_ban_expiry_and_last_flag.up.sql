ALTER TABLE bans RENAME COLUMN banned_until TO expires_at;
ALTER TABLE bans ADD COLUMN last_flagged_at timestamptz;

ALTER TABLE account_bans RENAME COLUMN banned_until TO expires_at;
ALTER TABLE account_bans ADD COLUMN last_flagged_at timestamptz;
