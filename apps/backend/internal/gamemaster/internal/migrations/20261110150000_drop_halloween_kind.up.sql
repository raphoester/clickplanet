-- The one Halloween entry (01a19262-2e9f-7f1b-bad5-99ab86c60237, 2026-10-31) plays on as a plain one: its kind only dressed the page.
UPDATE schedule_entries SET kind = 'plain' WHERE kind = 'halloween';
ALTER TABLE schedule_entries DROP CONSTRAINT schedule_entries_times;
ALTER TYPE entry_kind RENAME TO entry_kind_before;
CREATE TYPE entry_kind AS ENUM ('plain', 'battle');
ALTER TABLE schedule_entries ALTER COLUMN kind TYPE entry_kind USING kind::text::entry_kind;
DROP TYPE entry_kind_before;
ALTER TABLE schedule_entries ADD CONSTRAINT schedule_entries_times
    CHECK ((kind = 'battle' AND times BETWEEN 1 AND 10) OR (kind = 'plain' AND times = 0));
