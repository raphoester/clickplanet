-- One row per account. A guest is an account no provider is linked to yet.
CREATE TABLE accounts (
    id           uuid        PRIMARY KEY,
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL
);

-- One row per signed-in browser. Only the hash of the cookie's token is kept, so a copy of this table signs nobody in.
CREATE TABLE sessions (
    token_hash  bytea       PRIMARY KEY,
    account_id  uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL,
    extended_at timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL
);
