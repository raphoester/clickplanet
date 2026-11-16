UPDATE rule_sets SET rules = rules #- '{charges,rally}';

ALTER TABLE charges DROP COLUMN rallies;
