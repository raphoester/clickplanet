-- A hand holds one virus at most, as a bomb.
ALTER TABLE charges ADD COLUMN virus boolean NOT NULL DEFAULT false;

-- A virus makes 32 infections at most, its first included, and spreads every 20 s.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,virus}', '{"infections": 32, "every": "20s"}')
WHERE name = 'default';
