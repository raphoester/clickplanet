-- The name each player chose. An account with no row has no name. No foreign key: accounts live in the
-- auth schema, and the module hears auth.v1.AccountDeleted instead.
CREATE TABLE profiles (
    account_id uuid        PRIMARY KEY,
    name       text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 24),
    updated_at timestamptz NOT NULL
);

-- What each player did on the map. An account with no row never took a tile.
-- The streak is days in a row, UTC, with at least one take, ending on streak_last_day.
CREATE TABLE stats (
    account_id      uuid   PRIMARY KEY,
    tiles_taken     bigint NOT NULL CHECK (tiles_taken >= 0),
    streak_current  bigint NOT NULL CHECK (streak_current >= 0),
    streak_best     bigint NOT NULL CHECK (streak_best >= streak_current),
    streak_last_day date   NOT NULL
);
