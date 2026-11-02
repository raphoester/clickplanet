DELETE FROM outbox WHERE event IN ('auth.v1.AccountDeletionRequested', 'auth.v1.AccountRestored');

DROP INDEX accounts_deletion_requested_at;

ALTER TABLE accounts DROP COLUMN deletion_requested_at;
