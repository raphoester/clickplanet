-- No box and no right answer is a virus until the levels open it.
UPDATE rule_sets SET rules = jsonb_set(rules, '{kinds,virus}', '0') WHERE name = 'default';
