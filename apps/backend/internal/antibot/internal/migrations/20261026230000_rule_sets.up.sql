CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    -- An antibot.v1.Rules, as protojson writes it: every value of the set, its zeros too.
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- Production's rules on 26 October 2026: deploy/vps/backend.yaml, with what each watchdog filled in for the keys it
-- left out.
INSERT INTO rule_sets (name, rules) VALUES ('default', '
    {
      "bans": {
        "enforce": true,
        "banDurations": [
          "86400s",
          "604800s",
          "94608000s"
        ],
        "reflagInterval": "300s"
      },
      "jury": {
        "minSuspects": 2,
        "suspicionWindow": "600s"
      },
      "retaker": {
        "enabled": true,
        "reactionWindow": "5s",
        "minReactions": 20,
        "maxSpread": "0.300s",
        "maxMedian": "0.250s",
        "minTiles": 15,
        "roamMedian": "0.600s",
        "certainTiles": 0
      },
      "sequencer": {
        "enabled": true,
        "minSteps": 40,
        "minShare": 0.75,
        "certainSteps": 200,
        "certainShare": 0.95
      },
      "metronome": {
        "enabled": true,
        "maxGap": "7s",
        "maxSpread": "0.400s",
        "minClicks": 120,
        "certainFor": "1800s",
        "certainClicks": 900,
        "shape": {
          "maxGap": "10s",
          "clicks": 500,
          "certainClicks": 1000
        },
        "clock": {
          "period": "1s",
          "clicks": 120,
          "certainClicks": 600,
          "certainFor": "1800s"
        },
        "stamina": {
          "clicks": 40,
          "window": "21600s",
          "minBusy": "19800s",
          "certainBusy": "0s"
        }
      },
      "defender": {
        "enabled": true,
        "retakeWindow": "120s",
        "minClicks": 40,
        "minShare": 0.6,
        "certainClicks": 200,
        "certainShare": 0
      },
      "catcher": {
        "enabled": true,
        "minCatches": 5,
        "maxMedian": "3s",
        "certainMedian": "1.500s",
        "foreign": {
          "window": "3600s",
          "minClaims": 0,
          "certainClaims": 0
        }
      },
      "cohort": {
        "enabled": true,
        "startWindow": "5s",
        "minClicks": 20,
        "minFlagShare": 0.9,
        "rateRatio": 1.5,
        "lengthRatio": 1.2,
        "quietAfter": "60s",
        "minMembers": 2,
        "certainCohorts": 3,
        "certainMembers": 6,
        "chainWindow": "1800s"
      },
      "scraper": {
        "enabled": true,
        "minMaps": 5,
        "certainMaps": 15,
        "certainOffMap": 2
      },
      "churner": {
        "enabled": true,
        "window": "3600s",
        "v6": {
          "minAccounts": 4,
          "certainAccounts": 6
        },
        "v4": {
          "minAccounts": 10,
          "certainAccounts": 20
        },
        "relay": {
          "handoff": "90s",
          "maxLife": "300s",
          "minClicks": 20,
          "minFlagShare": 0.9,
          "minLinks": 3,
          "certainLinks": 6
        }
      },
      "hopper": {
        "enabled": true,
        "minAngle": 45,
        "minGap": "0.100s",
        "maxGap": "30s",
        "minSteps": 40,
        "minShare": 0.4,
        "certainSteps": 200,
        "certainShare": 0
      },
      "rotator": {
        "enabled": true,
        "window": "86400s",
        "minSeconds": 15,
        "minByChance": 90,
        "maxShare": 0.05,
        "minSlots": 24,
        "certainSlots": 0
      }
    }
');
