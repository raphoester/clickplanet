CREATE TABLE titles (
    account_id uuid        NOT NULL,
    title      text        NOT NULL CHECK (title ~ '^[a-z_]+$'),
    earned_at  timestamptz NOT NULL,
    PRIMARY KEY (account_id, title)
);

CREATE TABLE title_backfills (
    title         text        PRIMARY KEY CHECK (title ~ '^[a-z_]+$'),
    backfilled_at timestamptz NOT NULL
);
