CREATE TYPE rule_change_source AS ENUM ('internal', 'operator');

CREATE TABLE rule_changes (
    request_id text               PRIMARY KEY CHECK (request_id <> ''),
    rule_set   text               NOT NULL CHECK (rule_set <> ''),
    source     rule_change_source NOT NULL,
    applied_at timestamptz        NOT NULL,
    seq        bigint             GENERATED ALWAYS AS IDENTITY
);

CREATE INDEX rule_changes_latest ON rule_changes (applied_at DESC, seq DESC);
