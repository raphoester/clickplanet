CREATE INDEX ledger_events_account ON ledger_events (account) WHERE account IS NOT NULL;

DROP INDEX ledger_events_account_kind;
