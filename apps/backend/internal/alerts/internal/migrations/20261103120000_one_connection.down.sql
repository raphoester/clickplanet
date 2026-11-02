ALTER TABLE roles DROP COLUMN color;

CREATE TABLE link_flows (
    account_id uuid        PRIMARY KEY,
    state      text        NOT NULL,
    verifier   text        NOT NULL,
    expires_at timestamptz NOT NULL
);
