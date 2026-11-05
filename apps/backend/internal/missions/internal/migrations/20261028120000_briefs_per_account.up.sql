ALTER TABLE mission_briefs RENAME TO flag_briefs;

CREATE TABLE mission_briefs (
    day        date  NOT NULL,
    account_id uuid  NOT NULL,
    missions   jsonb NOT NULL CHECK (jsonb_typeof(missions) = 'array'),
    PRIMARY KEY (day, account_id)
);

CREATE INDEX mission_briefs_account ON mission_briefs (account_id);

INSERT INTO mission_briefs (day, account_id, missions)
SELECT sheets.day, sheets.account_id, briefs.missions
FROM mission_sheets sheets JOIN flag_briefs briefs USING (day, flag);

DROP TABLE flag_briefs;

ALTER TABLE mission_sheets DROP COLUMN flag;
