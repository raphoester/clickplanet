-- IF NOT EXISTS: a run killed mid-build leaves the version dirty, and forcing it back runs this again.
CREATE INDEX IF NOT EXISTS ledger_events_account_kind ON ledger_events (account, kind, country, previous)
    WHERE account IS NOT NULL;

DROP INDEX IF EXISTS ledger_events_account;
