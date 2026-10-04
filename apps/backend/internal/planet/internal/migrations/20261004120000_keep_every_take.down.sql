DROP INDEX ledger_takes_account;

DELETE FROM ledger_takes WHERE scope IS NULL;
ALTER TABLE ledger_takes ALTER COLUMN scope SET NOT NULL;
