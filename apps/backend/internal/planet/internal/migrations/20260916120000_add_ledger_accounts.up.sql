ALTER TABLE ledger_takes ADD COLUMN account uuid;

CREATE TABLE ledger_forgotten_accounts (
    account         uuid   PRIMARY KEY,
    before_position bigint NOT NULL CHECK (before_position >= 0)
);
