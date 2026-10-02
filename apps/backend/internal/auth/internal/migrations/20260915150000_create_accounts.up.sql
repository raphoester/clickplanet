CREATE TABLE accounts (
    id           uuid        PRIMARY KEY,
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL
);

CREATE TABLE sessions (
    token_hash  bytea       PRIMARY KEY,
    account_id  uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL,
    extended_at timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL
);
