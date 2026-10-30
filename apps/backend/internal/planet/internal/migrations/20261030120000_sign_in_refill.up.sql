-- A guest's first sign-in earns a refill: on in every set kept, as rules.Snapshot.Restore reads a set with every value.
UPDATE rule_sets SET rules = jsonb_set(rules, '{bonuses,signInRefill}', 'true');
