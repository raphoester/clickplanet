UPDATE rule_sets SET rules = jsonb_set(rules, '{siege}', '{"on": false, "freeze": "8s"}') WHERE name = 'default';
