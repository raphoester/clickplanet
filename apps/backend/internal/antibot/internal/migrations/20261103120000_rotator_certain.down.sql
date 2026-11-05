UPDATE rule_sets SET rules = jsonb_set(rules, '{rotator,certainSlots}', '0') WHERE name = 'default';
UPDATE rule_sets SET rules = rules #- '{rotator,certainByChance}';
