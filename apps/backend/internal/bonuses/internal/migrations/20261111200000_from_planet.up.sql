-- A set is a bonuses.v1.Rules as protojson: rules.Snapshot.Restore is the check. "default" holds every value, another
-- set what it changes.
CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- The rule set the schedule told the offers to play last, which they play at boot. One row at most.
CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    decided_at timestamptz NOT NULL
);

-- Planet kept the offers' half of each of its sets under "bonuses" until the offers were a module of their own: each
-- set comes across under its name, with what it held of that half and its "frozen". Planet's migration of the same
-- release leaves "bonuses" in its sets, so this reads it whichever module migrates first. A value a set held as null,
-- which read as unset, is left out.
DO $$
BEGIN
    IF to_regclass('planet.rule_sets') IS NOT NULL THEN
        INSERT INTO rule_sets (name, rules)
        SELECT name, jsonb_strip_nulls(jsonb_build_object(
            'boxes', b -> 'boxes',
            'kinds', b -> 'kinds',
            'offerTtl', b -> 'offerTtl',
            'missRetry', b -> 'missRetry',
            'activeWithin', b -> 'activeWithin',
            'spread', CASE WHEN b -> 'spread' ? 'maxPerBox'
                THEN jsonb_build_object('maxPerBox', b -> 'spread' -> 'maxPerBox') END,
            'enclose', CASE WHEN b -> 'enclose' ? 'maxPerBox'
                THEN jsonb_build_object('maxPerBox', b -> 'enclose' -> 'maxPerBox') END,
            'shield', CASE WHEN b -> 'shield' ?| ARRAY['minPerBox', 'maxPerBox']
                THEN jsonb_build_object('minPerBox', b -> 'shield' -> 'minPerBox', 'maxPerBox', b -> 'shield' -> 'maxPerBox') END,
            'welcome', b -> 'welcome',
            'quiz', b -> 'quiz',
            'signInRefill', b -> 'signInRefill',
            'frozen', played -> 'frozen'
        ))
        FROM (
            SELECT name, rules AS played, COALESCE(rules -> 'bonuses', '{}'::jsonb) AS b FROM planet.rule_sets
        ) kept;
    END IF;
END $$;

-- A database with no planet rules to copy: production's default of 11 November 2026, as planet's migrations seed it.
INSERT INTO rule_sets (name, rules) VALUES ('default', '{
    "boxes": {"minInterval": "240s", "maxInterval": "480s", "maxChargesPerHour": 12},
    "kinds": {"refill": 5, "spreadClicks": 3, "encloseClicks": 2, "bomb": 1, "shields": 3},
    "offerTtl": "15s", "missRetry": "120s", "activeWithin": "120s",
    "spread": {"maxPerBox": 4},
    "enclose": {"maxPerBox": 3},
    "shield": {"minPerBox": 5, "maxPerBox": 20},
    "welcome": {"within": "600s", "boxAfter": "60s"},
    "quiz": {
        "enabled": true, "minInterval": "240s", "maxInterval": "480s", "missRetry": "120s", "offerTtl": "25s",
        "answerWindow": "8s", "leaderBias": 1, "maxChargesPerHour": 12
    },
    "signInRefill": true,
    "frozen": false
}') ON CONFLICT (name) DO NOTHING;
