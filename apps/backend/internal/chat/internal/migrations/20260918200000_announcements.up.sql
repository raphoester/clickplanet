CREATE TABLE announcements (
    id           uuid        PRIMARY KEY,
    kind         text        NOT NULL,
    payload      jsonb       NOT NULL,
    announced_at timestamptz NOT NULL
);

CREATE INDEX announcements_announced_at ON announcements (announced_at);
