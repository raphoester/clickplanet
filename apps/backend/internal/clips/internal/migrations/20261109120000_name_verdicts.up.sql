CREATE TYPE verdict_by AS ENUM ('judge', 'operator');

-- Whether a player's name may be shown in the game's posts: one row a player, judged again once it renames.
CREATE TABLE name_verdicts (
    account    uuid        PRIMARY KEY,
    name       text        NOT NULL CHECK (name <> ''),
    shown      boolean     NOT NULL,
    reason     text        NOT NULL,
    decided_by verdict_by  NOT NULL,
    decided_at timestamptz NOT NULL
);

CREATE INDEX name_verdicts_decided_at ON name_verdicts (decided_at DESC);
