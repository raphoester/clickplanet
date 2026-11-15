-- A set is a boxes.v1.Rules as protojson: rules.Snapshot.Restore is the check. "default" holds every value, another set
-- what it changes.
CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- The rule set the schedule told the boxes to play last, which they play at boot. One row at most.
CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    decided_at timestamptz NOT NULL
);

-- The bonuses module kept the boxes' half of each of its sets ("boxes", "offerTtl", "missRetry", "activeWithin",
-- "welcome" and "frozen") until the boxes were a module of their own: each set comes across under its name. The bonuses
-- module's migrations of the same release leave its sets as they are, so this reads them whichever module migrates
-- first. A value a set held as null, which read as unset, is left out.
DO $$
BEGIN
    IF to_regclass('bonuses.rule_sets') IS NOT NULL THEN
        INSERT INTO rule_sets (name, rules)
        SELECT name, jsonb_strip_nulls(jsonb_build_object(
            'minInterval', b -> 'minInterval',
            'maxInterval', b -> 'maxInterval',
            'maxBoxesPerHour', b -> 'maxChargesPerHour',
            'offerTtl', played -> 'offerTtl',
            'missRetry', played -> 'missRetry',
            'activeWithin', played -> 'activeWithin',
            'welcome', played -> 'welcome',
            'frozen', played -> 'frozen'
        ))
        FROM (
            SELECT name, rules AS played, COALESCE(rules -> 'boxes', '{}'::jsonb) AS b FROM bonuses.rule_sets
        ) kept;
    END IF;
END $$;

-- A database with no bonuses rules to copy: production's box half of 15 November 2026.
INSERT INTO rule_sets (name, rules) VALUES ('default', '{
    "minInterval": "240s", "maxInterval": "480s", "maxBoxesPerHour": 12,
    "offerTtl": "15s", "missRetry": "120s", "activeWithin": "120s",
    "welcome": {"within": "600s", "boxAfter": "60s"},
    "frozen": false
}') ON CONFLICT (name) DO NOTHING;
