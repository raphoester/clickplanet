-- A battle's points' times, 1 to 10; a plain entry's are 0. The only battle planned is the Final Battle, worth three days.
ALTER TABLE schedule_entries ADD COLUMN times bigint NOT NULL DEFAULT 0;
UPDATE schedule_entries SET times = 3 WHERE kind = 'battle';
ALTER TABLE schedule_entries ALTER COLUMN times DROP DEFAULT;
ALTER TABLE schedule_entries ADD CONSTRAINT schedule_entries_times
    CHECK ((kind = 'battle' AND times BETWEEN 1 AND 10) OR (kind = 'plain' AND times = 0));
