UPDATE rule_sets SET rules = rules #- '{charges,virus}';

ALTER TABLE charges DROP COLUMN virus;
