ALTER TABLE ledger_takes RENAME TO ledger_events;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_takes_pkey TO ledger_events_pkey;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_takes_position_check TO ledger_events_position_check;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_takes_tile_check TO ledger_events_tile_check;
ALTER INDEX ledger_takes_account RENAME TO ledger_events_account;

ALTER TABLE ledger_events ADD COLUMN kind text NOT NULL DEFAULT 'take' CHECK (kind <> '');
ALTER TABLE ledger_events ALTER COLUMN kind DROP DEFAULT;

ALTER TABLE ledger_events ADD COLUMN payload jsonb;
