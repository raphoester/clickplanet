DELETE FROM stats WHERE streak_last_day IS NULL;
ALTER TABLE stats ALTER COLUMN streak_last_day SET NOT NULL;
ALTER TABLE stats DROP COLUMN messages_sent;
