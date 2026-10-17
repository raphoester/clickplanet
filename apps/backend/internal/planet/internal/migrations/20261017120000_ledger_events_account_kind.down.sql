CREATE INDEX IF NOT EXISTS ledger_events_account ON ledger_events (account) WHERE account IS NOT NULL;

DROP INDEX IF EXISTS ledger_events_account_kind;
