-- The account that holds each flag: the one whose payment opened it or claimed it last. A flag opened by hand has none.
-- Each flag opened from a submission is held by its founder.
CREATE TABLE holders (
    flag       text PRIMARY KEY REFERENCES flags (id) ON DELETE CASCADE,
    account_id uuid NOT NULL
);

CREATE INDEX holders_account ON holders (account_id);

INSERT INTO holders (flag, account_id)
SELECT flag, account_id FROM submissions WHERE state = 'opened';

-- The operator's close is a removal: each flag closed with a note is removed. A flag closed at its end, or before the
-- notes existed, stays closed, free for a player to claim.
UPDATE flags SET state = 'removed' WHERE state = 'closed' AND id IN (SELECT flag FROM closures);

-- A removed flag's name is free for a new flag.
DROP INDEX flags_name_key;
CREATE UNIQUE INDEX flags_name_key ON flags (lower(name)) WHERE state <> 'removed';
