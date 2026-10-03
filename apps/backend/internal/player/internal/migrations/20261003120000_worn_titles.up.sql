CREATE TABLE worn_titles (
    account_id uuid        PRIMARY KEY,
    title      text        NOT NULL CHECK (title ~ '^[a-z_]+$'),
    worn_at    timestamptz NOT NULL
);
