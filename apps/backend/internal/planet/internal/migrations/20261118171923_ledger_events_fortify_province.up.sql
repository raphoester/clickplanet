-- The last fortify of a province, for the territory card. A fortify before provinces names a landmass, which nothing
-- reads by any more. IF NOT EXISTS: a run killed mid-build leaves the version dirty, and forcing it back runs this again.
DROP INDEX IF EXISTS ledger_events_fortify;
CREATE INDEX IF NOT EXISTS ledger_events_fortify_province ON ledger_events (((payload ->> 'province')::integer), position)
    WHERE kind = 'fortify';
