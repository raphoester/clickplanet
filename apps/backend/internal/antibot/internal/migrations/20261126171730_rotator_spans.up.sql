UPDATE rule_sets
SET rules = jsonb_set(rules, '{rotator}', (rules -> 'rotator') - 'minSlots' - 'certainSlots' || '{
    "span": "60s",
    "minSeconds": 3,
    "minByChance": 15,
    "minSpans": 10,
    "certainSpans": 10,
    "certainByChance": 30
  }')
WHERE name = 'default';

UPDATE rule_sets
SET rules = rules #- '{rotator,minSlots}' #- '{rotator,certainSlots}'
WHERE name <> 'default' AND rules -> 'rotator' ?| ARRAY['minSlots', 'certainSlots'];
