-- IF NOT EXISTS: a run killed mid-build leaves the version dirty, and forcing it back runs this again.
CREATE INDEX IF NOT EXISTS ledger_events_fortify ON ledger_events (((payload ->> 'landmass')::integer), position)
    WHERE kind = 'fortify';
