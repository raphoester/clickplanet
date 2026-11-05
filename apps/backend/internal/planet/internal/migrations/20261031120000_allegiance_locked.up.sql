-- The flags stay open in every set kept, as rules.Snapshot.Restore reads a set with every value.
UPDATE rule_sets SET rules = jsonb_set(rules, '{allegianceLocked}', 'false');
