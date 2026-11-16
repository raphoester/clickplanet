UPDATE rule_sets SET rules = rules #- '{charges,mine}';

ALTER TABLE charges DROP COLUMN mines;
