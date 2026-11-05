-- Enclose spends a charge, until a set says otherwise.
UPDATE rule_sets SET rules = jsonb_set(rules, '{bonuses,enclose,unlimited}', 'false') WHERE name = 'default';
