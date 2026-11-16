UPDATE rule_sets SET rules = rules #- '{charges,refill}' #- '{charges,bomb,held}' #- '{charges,virus,held}';

ALTER TABLE charges RENAME COLUMN refills TO refill;
ALTER TABLE charges RENAME COLUMN bombs TO bomb;
ALTER TABLE charges RENAME COLUMN viruses TO virus;
ALTER TABLE charges DROP CONSTRAINT charges_refills_check,
    ALTER COLUMN refill DROP DEFAULT,
    ALTER COLUMN refill TYPE boolean USING refill > 0,
    ALTER COLUMN refill SET DEFAULT false;
ALTER TABLE charges DROP CONSTRAINT charges_bombs_check,
    ALTER COLUMN bomb DROP DEFAULT,
    ALTER COLUMN bomb TYPE boolean USING bomb > 0,
    ALTER COLUMN bomb SET DEFAULT false;
ALTER TABLE charges DROP CONSTRAINT charges_viruses_check,
    ALTER COLUMN virus DROP DEFAULT,
    ALTER COLUMN virus TYPE boolean USING virus > 0,
    ALTER COLUMN virus SET DEFAULT false;
