-- Every username an operator gave an account, kept for player.renameRetention.
CREATE TABLE renames (
    id          bigserial   PRIMARY KEY,
    account_id  uuid        NOT NULL,
    renamed_at  timestamptz NOT NULL,
    old_name    text        NOT NULL,
    new_name    text        NOT NULL,
    reason      text        NOT NULL,
    -- NULL for a call on the console listener, and once the operator's account is forgotten.
    operator_id uuid,
    client      text        NOT NULL
);

CREATE INDEX renames_account_idx ON renames (account_id, renamed_at DESC);

CREATE INDEX renames_renamed_at_idx ON renames (renamed_at DESC);
