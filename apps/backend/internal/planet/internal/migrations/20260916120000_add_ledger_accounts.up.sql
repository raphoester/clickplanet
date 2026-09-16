-- The account a take's click token named. NULL for none, and for every take made before accounts.
ALTER TABLE ledger_takes ADD COLUMN account uuid;

-- A reverted account's takes before before_position are forgotten, whatever scope they came from.
CREATE TABLE ledger_forgotten_accounts (
    account         uuid   PRIMARY KEY,
    before_position bigint NOT NULL CHECK (before_position >= 0)
);
