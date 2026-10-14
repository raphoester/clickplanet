CREATE TABLE uploads (
    id         uuid        PRIMARY KEY,
    size_bytes bigint      NOT NULL,
    granted_at timestamptz NOT NULL
);

CREATE INDEX attachments_id ON attachments (id);
