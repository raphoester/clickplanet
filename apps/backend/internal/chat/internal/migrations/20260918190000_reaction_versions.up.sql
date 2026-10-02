CREATE TABLE reaction_versions (
    message_id text        PRIMARY KEY,
    version    bigint      NOT NULL,
    changed_at timestamptz NOT NULL
);

CREATE INDEX reaction_versions_changed_at ON reaction_versions (changed_at);
