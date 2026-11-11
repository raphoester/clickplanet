-- The difficulty-weighted draw, off: width 0 is the draw of before. The rest is what it plays once switched on.
UPDATE rule_sets SET rules = jsonb_set(rules, '{quiz}', COALESCE(rules -> 'quiz', '{}'::jsonb) || '{
    "target": 0.7, "width": 0, "floor": 0.42, "floorAfter": 15, "explore": 0.1
}'::jsonb) WHERE name = 'default';
