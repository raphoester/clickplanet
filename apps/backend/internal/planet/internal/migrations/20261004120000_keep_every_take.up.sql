ALTER TABLE ledger_takes ALTER COLUMN scope DROP NOT NULL;

CREATE INDEX ledger_takes_account ON ledger_takes (account) WHERE account IS NOT NULL;
