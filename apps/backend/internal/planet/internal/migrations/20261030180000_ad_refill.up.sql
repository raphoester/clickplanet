-- A rewarded ad fills the bank: on in every set kept, as rules.Snapshot.Restore reads a set with every value.
UPDATE rule_sets SET rules = jsonb_set(rules, '{bonuses,adRefill}', '{"enabled": true, "every": "3600s"}');
