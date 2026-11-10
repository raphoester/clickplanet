UPDATE rule_sets
SET rules = jsonb_set(jsonb_set(rules, '{arena}', coalesce(rules -> 'arena', '{}')), '{arena,capitalShields}',
                      rules #> '{seed,capitals,shields}')
WHERE rules #> '{seed,capitals,shields}' IS NOT NULL;

UPDATE rule_sets SET rules = rules - 'seed' - 'reach' - 'capitals';
