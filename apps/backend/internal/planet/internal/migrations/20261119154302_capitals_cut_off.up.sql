-- A take that cuts a flag's land from its capital clears the land cut off while a set holds capitals.cutOff. Off in
-- default.
UPDATE rule_sets SET rules = jsonb_set(rules, '{capitals,cutOff}', 'false') WHERE name = 'default';
