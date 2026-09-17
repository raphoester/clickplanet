-- One row per provider user linked to an account. An account with none is a guest.
CREATE TABLE identities (
    provider       text        NOT NULL,
    subject        text        NOT NULL,
    account_id     uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    -- Only an address the provider says is verified. Never used to find or merge an account.
    email          text,
    email_verified boolean     NOT NULL,
    linked_at      timestamptz NOT NULL,
    PRIMARY KEY (provider, subject)
);

CREATE INDEX identities_account_id ON identities (account_id);

-- Sign-out everywhere and the cascade from a deleted account find sessions by account.
CREATE INDEX sessions_account_id ON sessions (account_id);

-- The guest prune finds idle accounts.
CREATE INDEX accounts_last_seen_at ON accounts (last_seen_at);
