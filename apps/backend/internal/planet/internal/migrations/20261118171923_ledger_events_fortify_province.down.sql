DROP INDEX IF EXISTS ledger_events_fortify_province;
CREATE INDEX IF NOT EXISTS ledger_events_fortify ON ledger_events (((payload ->> 'landmass')::integer), position)
    WHERE kind = 'fortify';
