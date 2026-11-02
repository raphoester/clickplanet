ALTER TABLE accounts ADD COLUMN deletion_requested_at timestamptz;

CREATE INDEX accounts_deletion_requested_at ON accounts (deletion_requested_at) WHERE deletion_requested_at IS NOT NULL;

ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'auth.v1.AccountDeletionRequested';
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'auth.v1.AccountRestored';
