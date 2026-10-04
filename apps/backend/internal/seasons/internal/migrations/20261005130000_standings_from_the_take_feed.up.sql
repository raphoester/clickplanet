CREATE TABLE standings_position (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    position  bigint  NOT NULL CHECK (position >= 0)
);
