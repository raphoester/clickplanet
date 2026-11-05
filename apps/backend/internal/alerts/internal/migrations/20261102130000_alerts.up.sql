CREATE TABLE link_flows (
    account_id uuid        PRIMARY KEY,
    state      text        NOT NULL,
    verifier   text        NOT NULL,
    expires_at timestamptz NOT NULL
);

-- flag NULL: the account never pledged one, and its first take gives it.
CREATE TABLE links (
    account_id   uuid        PRIMARY KEY,
    discord_user text        NOT NULL UNIQUE,
    flag         text,
    linked_at    timestamptz NOT NULL
);

CREATE TABLE roles (
    key     text PRIMARY KEY,
    role_id text NOT NULL
);

CREATE TABLE attack_pings (
    flag    text        NOT NULL,
    sent_at timestamptz NOT NULL,
    PRIMARY KEY (flag, sent_at)
);

CREATE TABLE start_pings (
    key     text        PRIMARY KEY,
    sent_at timestamptz NOT NULL
);
