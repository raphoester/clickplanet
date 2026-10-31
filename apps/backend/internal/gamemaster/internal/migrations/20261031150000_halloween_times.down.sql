UPDATE schedule_entries SET kind = 'plain' WHERE kind = 'halloween';
ALTER TABLE schedule_entries DROP CONSTRAINT IF EXISTS schedule_entries_times;
ALTER TABLE schedule_entries ADD CONSTRAINT schedule_entries_times
    CHECK ((kind = 'battle' AND times BETWEEN 1 AND 10) OR (kind = 'plain' AND times = 0));
