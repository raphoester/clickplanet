-- No enclosure while a set holds charges.enclose.off: no hand has room for one, and a click with enclose on closes
-- nothing. Off in default.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,enclose,off}', 'false') WHERE name = 'default';
