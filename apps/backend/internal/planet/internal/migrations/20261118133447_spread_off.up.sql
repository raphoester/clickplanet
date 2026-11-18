-- No spread while a set holds charges.spread.off: a click with spread on takes its tile alone and spends nothing, and
-- a grant keeps none. Off in default.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,spread,off}', 'false') WHERE name = 'default';
