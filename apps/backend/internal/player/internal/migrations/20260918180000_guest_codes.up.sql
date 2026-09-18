-- The code the game shows for an account with no username, as "guest_" and the code. Drawn the first time the
-- account is shown and kept until it is deleted. No foreign key, as for profiles.
CREATE TABLE guest_codes (
    account_id uuid PRIMARY KEY,
    -- The store maps a violation of this constraint to players.ErrGuestCodeTaken, by its name.
    code       text NOT NULL CONSTRAINT guest_codes_code_key UNIQUE CHECK (code ~ '^[0-9a-f]{6}$')
);
