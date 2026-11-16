-- A hand holds rallies in a stack, as mines.
ALTER TABLE charges ADD COLUMN rallies integer NOT NULL DEFAULT 0;

-- A hand holds 3 rallies at most. A rally's ring is 18 tile spacings wide, about 1,000 tiles, and lasts 3 minutes, and a
-- click of its flag inside it takes half a click from the bank.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,rally}', '{"held": 3, "rings": 18, "lasts": "180s", "cost": 0.5}')
WHERE name = 'default';
