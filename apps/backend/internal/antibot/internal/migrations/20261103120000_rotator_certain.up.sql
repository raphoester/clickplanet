UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(rules, '{rotator,certainSlots}', '48'), '{rotator,certainByChance}', '300')
WHERE name = 'default';
