CREATE TYPE report_reason AS ENUM ('cheating', 'chat_abuse', 'bad_name', 'other');

CREATE TABLE reports (
    id                  uuid          PRIMARY KEY,
    reporter_account_id uuid          NOT NULL,
    reported_account_id uuid          NOT NULL,
    reason              report_reason NOT NULL,
    text                text          NOT NULL,
    message_id          text          NOT NULL,
    reported_at         timestamptz   NOT NULL
);

CREATE INDEX reports_reported_at ON reports (reported_at);

CREATE INDEX reports_reporter ON reports (reporter_account_id, reported_account_id);

CREATE INDEX reports_reported ON reports (reported_account_id);
