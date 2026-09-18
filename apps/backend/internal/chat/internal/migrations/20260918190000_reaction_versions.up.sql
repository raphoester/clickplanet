-- One counter per message, bumped in the same statement as each change to its reactions. A tally carries it, so a
-- client can drop a frame older than the one it holds: frames may be published in any order.
CREATE TABLE reaction_versions (
    message_id text        PRIMARY KEY,
    version    bigint      NOT NULL,
    changed_at timestamptz NOT NULL
);

-- The prune deletes by age, as for the reactions.
CREATE INDEX reaction_versions_changed_at ON reaction_versions (changed_at);
