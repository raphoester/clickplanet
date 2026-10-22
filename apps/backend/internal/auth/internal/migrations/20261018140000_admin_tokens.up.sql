CREATE TYPE admin_token_kind AS ENUM ('access', 'refresh');

CREATE TABLE admin_tokens (
    token_hash bytea            PRIMARY KEY,
    kind       admin_token_kind NOT NULL,
    account_id uuid             NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    client_id  text             NOT NULL,
    client     text             NOT NULL,
    created_at timestamptz      NOT NULL,
    expires_at timestamptz      NOT NULL
);

CREATE INDEX admin_tokens_expiry ON admin_tokens (expires_at);
