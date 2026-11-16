-- Every kind of charge is a count: a hand may hold several refills, bombs and viruses, as many as the rules in force
-- say (charges.refill.held, charges.bomb.held, charges.virus.held). A hand that held one holds 1.
ALTER TABLE charges ALTER COLUMN refill DROP DEFAULT,
    ALTER COLUMN refill TYPE integer USING refill::integer,
    ALTER COLUMN refill SET DEFAULT 0,
    ADD CONSTRAINT charges_refills_check CHECK (refill >= 0);
ALTER TABLE charges ALTER COLUMN bomb DROP DEFAULT,
    ALTER COLUMN bomb TYPE integer USING bomb::integer,
    ALTER COLUMN bomb SET DEFAULT 0,
    ADD CONSTRAINT charges_bombs_check CHECK (bomb >= 0);
ALTER TABLE charges ALTER COLUMN virus DROP DEFAULT,
    ALTER COLUMN virus TYPE integer USING virus::integer,
    ALTER COLUMN virus SET DEFAULT 0,
    ADD CONSTRAINT charges_viruses_check CHECK (virus >= 0);
ALTER TABLE charges RENAME COLUMN refill TO refills;
ALTER TABLE charges RENAME COLUMN bomb TO bombs;
ALTER TABLE charges RENAME COLUMN virus TO viruses;

-- A hand holds one of each, as before.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,refill}', '{"held": 1}')
WHERE name = 'default';
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,bomb,held}', '1')
WHERE name = 'default';
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,virus,held}', '1')
WHERE name = 'default';
