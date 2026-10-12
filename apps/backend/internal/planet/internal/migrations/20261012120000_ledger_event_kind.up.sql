CREATE TYPE ledger_event_kind AS ENUM ('take', 'strike', 'spread', 'enclose', 'bomb', 'shield', 'fortify');

ALTER TABLE ledger_events
    DROP CONSTRAINT ledger_events_kind_check,
    ALTER COLUMN kind TYPE ledger_event_kind USING kind::ledger_event_kind;
