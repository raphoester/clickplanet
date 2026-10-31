-- A Halloween entry scores no points, as a plain one. Apart from the enum's new value: a value is used only once committed.
ALTER TABLE schedule_entries DROP CONSTRAINT IF EXISTS schedule_entries_times;
ALTER TABLE schedule_entries ADD CONSTRAINT schedule_entries_times
    CHECK ((kind = 'battle' AND times BETWEEN 1 AND 10) OR (kind IN ('plain', 'halloween') AND times = 0));
