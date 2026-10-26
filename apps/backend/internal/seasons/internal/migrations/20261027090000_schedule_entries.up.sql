-- The operators' entries of the rule schedule; the seasons' phases are the config's.
CREATE TABLE schedule_entries (
    id        uuid        PRIMARY KEY,
    rule_set  text        NOT NULL CHECK (rule_set <> ''),
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL CHECK (ends_at > starts_at),
    priority  bigint      NOT NULL CHECK (priority > 0 AND priority <= 4294967295),
    made_at   timestamptz NOT NULL
);

CREATE INDEX schedule_entries_ends_at ON schedule_entries (ends_at);
