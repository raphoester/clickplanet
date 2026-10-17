CREATE INDEX ledger_events_account_kind ON ledger_events (account, kind, country, previous)
    WHERE account IS NOT NULL;

DROP INDEX ledger_events_account;
