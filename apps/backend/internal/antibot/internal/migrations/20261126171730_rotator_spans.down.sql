UPDATE rule_sets
SET rules = jsonb_set(rules, '{rotator}', (rules -> 'rotator') - 'span' - 'minSpans' - 'certainSpans' || '{
    "minSeconds": 15,
    "minByChance": 90,
    "minSlots": 24,
    "certainSlots": 48,
    "certainByChance": 300
  }')
WHERE name = 'default';

UPDATE rule_sets
SET rules = rules #- '{rotator,span}' #- '{rotator,minSpans}' #- '{rotator,certainSpans}'
WHERE name <> 'default';
