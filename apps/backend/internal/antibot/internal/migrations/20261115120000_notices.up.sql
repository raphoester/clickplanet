CREATE TABLE notices (
    account         uuid        PRIMARY KEY,
    alongside       uuid        NOT NULL,
    noticed_at      timestamptz NOT NULL,
    acknowledged_at timestamptz
);

UPDATE rule_sets
SET rules = jsonb_set(rules, '{sharing}', '{"enabled": true, "window": "3600s", "maxAccounts": 3}')
WHERE name = 'default';
