CREATE TABLE profiles (
    account_id uuid        PRIMARY KEY,
    name       text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 24),
    updated_at timestamptz NOT NULL
);

CREATE TABLE stats (
    account_id      uuid   PRIMARY KEY,
    tiles_taken     bigint NOT NULL CHECK (tiles_taken >= 0),
    streak_current  bigint NOT NULL CHECK (streak_current >= 0),
    streak_best     bigint NOT NULL CHECK (streak_best >= streak_current),
    streak_last_day date   NOT NULL
);
