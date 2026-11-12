-- A spread takes the tiles one step around the tile clicked and spends a charge, until a set says otherwise.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,spread}',
    coalesce(rules #> '{charges,spread}', '{}') || '{"rings": 1, "unlimited": false}')
WHERE name = 'default';
