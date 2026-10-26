-- A set is a planet.v1.RuleSet's JSON, with no name: every value written, read back by rules.Snapshot.Restore.
CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- The game's own rules on 26 October 2026: deploy/vps/backend.yaml's values before they moved here.
INSERT INTO rule_sets (name, rules) VALUES ('default', '{
    "tempo": 1,
    "boxes": {"minInterval": "240s", "maxInterval": "480s", "maxChargesPerHour": 12},
    "clicks": {
        "perSecond": 0.2, "burst": 60,
        "scopeMultiplier": 10, "v6ScopeMultiplier": 4, "guestScopeMultiplier": 1, "linkedMultiplier": 2,
        "newAccountClicks": 10
    },
    "tollSteps": [{"share": 0.1, "slowdown": 1.5}, {"share": 0.2, "slowdown": 2.5}, {"share": 0.3, "slowdown": 4}],
    "bonuses": {
        "kinds": {"refill": 5, "spreadClicks": 3, "encloseClicks": 2, "bomb": 1, "shields": 3},
        "offerTtl": "15s", "missRetry": "120s", "activeWithin": "120s",
        "spread": {"clicks": 8, "maxPerBox": 4},
        "bomb": {"rings": 4},
        "enclose": {"maxTiles": 25, "held": 3, "maxPerBox": 3},
        "shield": {"held": 30, "minPerBox": 5, "maxPerBox": 20, "perTile": 10},
        "welcome": {"within": "600s", "boxAfter": "60s"}
    },
    "quiz": {
        "enabled": true, "minInterval": "240s", "maxInterval": "480s", "missRetry": "120s", "offerTtl": "25s",
        "answerWindow": "8s", "leaderBias": 1, "maxChargesPerHour": 12
    }
}');
