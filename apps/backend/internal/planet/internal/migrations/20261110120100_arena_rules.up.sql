UPDATE rule_sets SET rules = jsonb_set(rules, '{arena}', '{"on": false, "freeze": "8s"}') WHERE name = 'default';
