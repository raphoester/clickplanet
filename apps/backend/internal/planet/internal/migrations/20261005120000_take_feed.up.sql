ALTER TABLE ledger_takes ADD COLUMN reverted boolean NOT NULL DEFAULT false;

UPDATE ledger_takes SET reverted = true
FROM ledger_forgotten
WHERE ledger_takes.scope = ledger_forgotten.scope
  AND ledger_takes.position < ledger_forgotten.before_position;

UPDATE ledger_takes SET reverted = true
FROM ledger_forgotten_accounts
WHERE ledger_takes.account = ledger_forgotten_accounts.account
  AND ledger_takes.position < ledger_forgotten_accounts.before_position
  AND ledger_takes.position >= COALESCE((SELECT head FROM ledger_head), 0);

CREATE TABLE ledger_feed (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    start     bigint  NOT NULL CHECK (start >= 0)
);

INSERT INTO ledger_feed (start)
SELECT GREATEST(COALESCE(max(position) + 1, 0), COALESCE((SELECT head FROM ledger_head), 0))
FROM ledger_takes;
