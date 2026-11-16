-- A hand holds mines in a stack, as enclosures.
ALTER TABLE charges ADD COLUMN mines integer NOT NULL DEFAULT 0;

-- A hand holds 3 mines at most, and a mine's blast takes 2 rings of tiles around its own.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,mine}', '{"held": 3, "rings": 2}')
WHERE name = 'default';
