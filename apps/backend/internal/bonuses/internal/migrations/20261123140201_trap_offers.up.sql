-- No box and no right answer is a trap until the operator weighs it: 0, closed to all while the gates are on, and 1 or
-- 2 traps a box once it weighs above 0.
UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(jsonb_set(rules, '{kinds,trap}', '0'), '{gates,open,trap}', 'false'),
    '{trap}', '{"maxPerBox": 2}')
WHERE name = 'default';
