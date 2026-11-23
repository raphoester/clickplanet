-- A hand holds traps in a stack, as mines.
ALTER TABLE charges ADD COLUMN traps integer NOT NULL DEFAULT 0;

-- A hand holds 3 traps at most, and a trap takes 10 clicks from the bank of the player whose take sprang it.
UPDATE rule_sets SET rules = jsonb_set(rules, '{charges,trap}', '{"held": 3, "clicks": 10}')
WHERE name = 'default';
