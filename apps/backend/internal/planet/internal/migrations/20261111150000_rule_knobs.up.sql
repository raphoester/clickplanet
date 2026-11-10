-- The shields each capital starts an arena with become the capitals seed's: a set's own value moves with it. The reach
-- and the capital rules become knobs of their own, off in default.
UPDATE rule_sets
SET rules = jsonb_set(rules #- '{arena,capitalShields}', '{seed}',
                      jsonb_build_object('capitals', jsonb_build_object('shields', rules #> '{arena,capitalShields}')))
WHERE rules #> '{arena,capitalShields}' IS NOT NULL;

UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(rules, '{reach}', 'false'), '{capitals}', '{"knockOut": false, "fixedShields": false}')
WHERE name = 'default';
