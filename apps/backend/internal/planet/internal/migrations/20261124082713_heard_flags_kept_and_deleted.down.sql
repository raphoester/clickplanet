DELETE FROM heard_flags WHERE state = 'deleted';

ALTER TABLE heard_flags DROP COLUMN kept;

ALTER TYPE heard_flag_state RENAME TO heard_flag_state_old;

CREATE TYPE heard_flag_state AS ENUM ('open', 'closed');

ALTER TABLE heard_flags ALTER COLUMN state TYPE heard_flag_state USING state::text::heard_flag_state;

DROP TYPE heard_flag_state_old;
