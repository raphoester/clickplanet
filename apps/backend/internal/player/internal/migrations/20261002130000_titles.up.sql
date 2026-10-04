CREATE TABLE titles (
    account_id uuid        NOT NULL,
    title      text        NOT NULL CHECK (title ~ '^[a-z_]+$'),
    earned_at  timestamptz NOT NULL,
    PRIMARY KEY (account_id, title)
);
