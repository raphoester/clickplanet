-- No box and no right answer is a rally until the operator weighs it: 0, closed to all while the gates are on, and one
-- rally a box once it weighs above 0.
UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(jsonb_set(rules, '{kinds,rally}', '0'), '{gates,open,rally}', 'false'),
    '{rally}', '{"maxPerBox": 1}')
WHERE name = 'default';
