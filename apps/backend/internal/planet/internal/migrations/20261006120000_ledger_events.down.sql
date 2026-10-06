DELETE FROM ledger_events WHERE kind <> 'take';

ALTER TABLE ledger_events DROP CONSTRAINT ledger_events_payload_check;
ALTER TABLE ledger_events DROP COLUMN payload;
ALTER TABLE ledger_events DROP COLUMN kind;

ALTER INDEX ledger_events_account RENAME TO ledger_takes_account;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_events_tile_check TO ledger_takes_tile_check;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_events_position_check TO ledger_takes_position_check;
ALTER TABLE ledger_events RENAME CONSTRAINT ledger_events_pkey TO ledger_takes_pkey;
ALTER TABLE ledger_events RENAME TO ledger_takes;
