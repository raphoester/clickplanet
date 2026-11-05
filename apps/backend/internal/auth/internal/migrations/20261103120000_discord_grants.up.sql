CREATE TABLE discord_grants (
    grant_id        uuid        PRIMARY KEY,
    account_id      uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    discord_user_id text        NOT NULL,
    sealed_token    bytea       NOT NULL,
    expires_at      timestamptz NOT NULL
);

CREATE INDEX discord_grants_account_id ON discord_grants (account_id);

CREATE INDEX discord_grants_expires_at ON discord_grants (expires_at);

ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'auth.v1.DiscordJoinGranted';
