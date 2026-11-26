UPDATE flags SET state = 'closed' WHERE state = 'removed';

DROP INDEX flags_open_ends_at;
DROP INDEX flags_name_key;

ALTER TYPE flag_state RENAME TO flag_state_old;

CREATE TYPE flag_state AS ENUM ('open', 'closed');

ALTER TABLE flags ALTER COLUMN state TYPE flag_state USING state::text::flag_state;

DROP TYPE flag_state_old;

CREATE INDEX flags_open_ends_at ON flags (ends_at) WHERE state = 'open';
CREATE UNIQUE INDEX flags_name_key ON flags (lower(name));
