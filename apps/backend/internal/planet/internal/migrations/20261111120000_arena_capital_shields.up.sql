UPDATE rule_sets SET rules = jsonb_set(rules, '{arena,capitalShields}', '100') WHERE name = 'default';
