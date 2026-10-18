CREATE TYPE report_outcome AS ENUM ('acted', 'dismissed');

CREATE TABLE report_decisions (
    report_id  uuid           PRIMARY KEY REFERENCES reports (id) ON DELETE CASCADE,
    outcome    report_outcome NOT NULL,
    comment    text           NOT NULL,
    decided_at timestamptz    NOT NULL
);
