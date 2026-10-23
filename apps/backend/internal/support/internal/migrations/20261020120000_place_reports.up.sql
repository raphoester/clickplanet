CREATE TYPE place_report_reason AS ENUM ('hate', 'sexual', 'other');

CREATE TABLE place_reports (
    id                  uuid                PRIMARY KEY,
    reporter_account_id uuid,
    tile_id             bigint              NOT NULL CHECK (tile_id > 0),
    reason              place_report_reason NOT NULL,
    text                text                NOT NULL,
    created_at          timestamptz         NOT NULL
);

CREATE INDEX place_reports_created_at ON place_reports (created_at);

CREATE INDEX place_reports_reporter ON place_reports (reporter_account_id);

CREATE TABLE place_report_decisions (
    report_id  uuid           PRIMARY KEY REFERENCES place_reports (id) ON DELETE CASCADE,
    outcome    report_outcome NOT NULL,
    comment    text           NOT NULL,
    decided_at timestamptz    NOT NULL
);
