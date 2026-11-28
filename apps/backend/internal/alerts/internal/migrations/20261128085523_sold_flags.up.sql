-- The flags opened since the game shipped, each by the name the latest of the flags module's facts told: the name its
-- role wears in the server.
CREATE TABLE sold_flags (
    flag     text        PRIMARY KEY,
    name     text        NOT NULL,
    named_at timestamptz NOT NULL
);

-- The name the bot gave a sold flag's role, so a rename renames it. Empty: a role of a flag the game ships with, named
-- once by the library and never renamed, or one made before the names were kept.
ALTER TABLE roles ADD COLUMN name text NOT NULL DEFAULT '';
