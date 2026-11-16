-- One level per linked account, from the XP its stats give. Only the level is kept: the XP is computed again at each
-- read, so a change of weights is one reconciliation.
CREATE TABLE IF NOT EXISTS levels (
    account_id uuid        PRIMARY KEY,
    level      integer     NOT NULL CHECK (level >= 1),
    reached_at timestamptz NOT NULL
);
