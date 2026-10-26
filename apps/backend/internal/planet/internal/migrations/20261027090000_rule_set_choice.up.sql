-- The rule set the schedule told planet to play last, which planet plays at boot. One row at most.
CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    decided_at timestamptz NOT NULL
);
