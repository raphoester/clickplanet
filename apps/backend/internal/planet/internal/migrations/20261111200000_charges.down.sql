-- "bonuses" still holds the wallet's half of a set kept before the up migration. A set saved since holds it under
-- "charges" alone, and loses it here: the bonuses module keeps the offers' half either way.
UPDATE rule_sets SET rules = rules - 'charges';
