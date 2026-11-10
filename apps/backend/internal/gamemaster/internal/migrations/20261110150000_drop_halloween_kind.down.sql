ALTER TABLE schedule_entries DROP CONSTRAINT schedule_entries_times;
ALTER TYPE entry_kind RENAME TO entry_kind_before;
CREATE TYPE entry_kind AS ENUM ('plain', 'battle', 'halloween');
ALTER TABLE schedule_entries ALTER COLUMN kind TYPE entry_kind USING kind::text::entry_kind;
DROP TYPE entry_kind_before;
ALTER TABLE schedule_entries ADD CONSTRAINT schedule_entries_times
    CHECK ((kind = 'battle' AND times BETWEEN 1 AND 10) OR (kind IN ('plain', 'halloween') AND times = 0));
