-- No rally while a set holds charges.rally.off: a call is refused and the rally stays held, and a grant keeps none. Off
-- in default.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,rally,off}', 'false') WHERE name = 'default';
