-- A new account refills at the signed-in pace for its first 30 minutes.
UPDATE rule_sets SET rules = jsonb_set(rules, '{clicks,fastStart}', '"1800s"') WHERE name = 'default';
