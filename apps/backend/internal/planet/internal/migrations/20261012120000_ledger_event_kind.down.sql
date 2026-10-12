ALTER TABLE ledger_events
    ALTER COLUMN kind TYPE text,
    ADD CONSTRAINT ledger_events_kind_check CHECK (kind <> '');

DROP TYPE ledger_event_kind;
