-- No bomb while a set holds bonuses.bomb.off: no box or quiz offers one, and a bomb held is not dropped. Off in default.
UPDATE rule_sets SET rules = jsonb_set(rules, '{bonuses,bomb,off}', 'false') WHERE name = 'default';
