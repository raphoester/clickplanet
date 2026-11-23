UPDATE rule_sets SET rules = rules #- '{charges,trap}';

ALTER TABLE charges DROP COLUMN traps;
