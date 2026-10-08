ALTER TABLE bans ADD COLUMN next_flag_at timestamptz;
ALTER TABLE account_bans ADD COLUMN next_flag_at timestamptz;
