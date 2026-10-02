CREATE TABLE guest_codes (
    account_id uuid PRIMARY KEY,
    code       text NOT NULL CONSTRAINT guest_codes_code_key UNIQUE CHECK (code ~ '^[0-9a-f]{6}$')
);
