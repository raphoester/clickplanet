CREATE TABLE stats_position (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    start     bigint  NOT NULL CHECK (start >= 0),
    position  bigint  NOT NULL CHECK (position >= start)
);

CREATE TABLE stats_baseline (
    account_id      uuid   PRIMARY KEY,
    tiles_taken     bigint NOT NULL CHECK (tiles_taken >= 0),
    streak_current  bigint NOT NULL CHECK (streak_current >= 0),
    streak_best     bigint NOT NULL CHECK (streak_best >= streak_current),
    streak_last_day date
);
