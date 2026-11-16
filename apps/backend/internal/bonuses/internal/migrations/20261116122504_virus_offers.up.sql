-- No box and no right answer is a virus until the levels open it: weighed 0, and closed to all while the gates are on.
UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(rules, '{kinds,virus}', '0'), '{gates,open,virus}', 'false')
WHERE name = 'default';
