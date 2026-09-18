-- A refill charge fills the account's click bank when the player chooses. NULL is none held.
ALTER TABLE charges ADD COLUMN refill_until timestamptz;
