-- A set is a seasons.v1.Rules as protojson, every value written: rules.Snapshot.Restore is the check.
CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- Days only: a round of its own is a set the operator makes, for a schedule entry to play.
INSERT INTO rule_sets (name, rules) VALUES ('default', '{"ownRound": {"enabled": false, "times": 1}}');

-- The rule set the schedule told seasons to play last, with the window of the entry that plays it, which seasons plays
-- at boot. One row at most.
CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    starts_at  timestamptz,
    ends_at    timestamptz,
    decided_at timestamptz NOT NULL,
    CHECK ((starts_at IS NULL) = (ends_at IS NULL))
);
