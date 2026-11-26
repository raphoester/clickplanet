DROP INDEX flags_name_key;
CREATE UNIQUE INDEX flags_name_key ON flags (lower(name));

UPDATE flags SET state = 'closed' WHERE state = 'removed';

DROP TABLE holders;
