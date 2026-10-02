CREATE TABLE identities (
    provider       text        NOT NULL,
    subject        text        NOT NULL,
    account_id     uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    email          text,
    email_verified boolean     NOT NULL,
    linked_at      timestamptz NOT NULL,
    PRIMARY KEY (provider, subject)
);

CREATE INDEX identities_account_id ON identities (account_id);

CREATE INDEX sessions_account_id ON sessions (account_id);

CREATE INDEX accounts_last_seen_at ON accounts (last_seen_at);
