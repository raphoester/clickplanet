ALTER TABLE stats ADD COLUMN messages_sent bigint NOT NULL DEFAULT 0 CHECK (messages_sent >= 0);
ALTER TABLE stats ALTER COLUMN streak_last_day DROP NOT NULL;
