-- The kinds progression unlocked for each account (bonuses.v1.KeepUnlocks): the latest decision only.
CREATE TYPE unlocked_kind AS ENUM ('refill', 'spread_clicks', 'enclose_clicks', 'bomb', 'shields');

CREATE TABLE unlocks (
    account_id uuid            PRIMARY KEY,
    kinds      unlocked_kind[] NOT NULL,
    decided_at timestamptz     NOT NULL
);

-- The gates stay off until the levels are shown: no draw changes. The other sets play default's.
UPDATE rule_sets
SET rules = jsonb_set(rules, '{gates}', '{
    "on": false,
    "open": {"refill": true, "spreadClicks": true, "encloseClicks": false, "bomb": false, "shields": true}
}')
WHERE name = 'default';
